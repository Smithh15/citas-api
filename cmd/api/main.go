package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/Smithh15/citas-api/docs"
	"github.com/Smithh15/citas-api/internal/config"
	"github.com/Smithh15/citas-api/internal/db"
	"github.com/Smithh15/citas-api/internal/db/sqlc"
	"github.com/Smithh15/citas-api/internal/handlers"
	"github.com/Smithh15/citas-api/internal/middleware"
)

// @title        Citas API
// @version      1.0
// @description  Sistema de reservas de citas médicas con control de concurrencia
// @host         localhost:8080
// @BasePath     /
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	ctx := context.Background()

	pgPool, err := db.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to postgres: %v", err)
	}
	defer pgPool.Close()

	redisClient, err := db.NewRedisClient(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("connecting to redis: %v", err)
	}
	defer redisClient.Close()

	queries := sqlc.New(pgPool)

	healthHandler := handlers.NewHealthHandler(pgPool, redisClient)
	authHandler := handlers.NewAuthHandler(queries, cfg.JWTSecret)
	availabilityHandler := &handlers.AvailabilityHandler{Queries: queries}
	appointmentHandler := &handlers.AppointmentHandler{Queries: queries}

	r := gin.Default()
	r.GET("/health", healthHandler.Check)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
	}

	protected := r.Group("/")
	protected.Use(middleware.RequireAuth(cfg.JWTSecret))
	{
		protected.GET("/me", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"user_id": c.GetString("userID"),
				"role":    c.GetString("role"),
			})
		})
		protected.GET("/appointments/available", appointmentHandler.GetAvailableSlots)

		doctorOnly := protected.Group("/")
		doctorOnly.Use(middleware.RequireRole("doctor"))
		{
			doctorOnly.GET("/doctor/ping", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "solo doctores ven esto"})
			})
			doctorOnly.POST("/doctor/availability", availabilityHandler.Create)
		}
	}

	addr := ":" + cfg.AppPort
	log.Printf("citas-api listening on %s (env=%s)", addr, cfg.AppEnv)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
