package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
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

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	defer asynqClient.Close()

	queries := sqlc.New(pgPool)

	healthHandler := handlers.NewHealthHandler(pgPool, redisClient)
	authHandler := handlers.NewAuthHandler(queries, cfg.JWTSecret)
	availabilityHandler := &handlers.AvailabilityHandler{Queries: queries}
	appointmentHandler := &handlers.AppointmentHandler{
		Queries:              queries,
		AsynqClient:          asynqClient,
		MinCancellationHours: cfg.MinCancellationHours,
	}

	r := gin.Default()
	// Gin confía en TODOS los proxies por defecto (0.0.0.0/0): sin esto,
	// cualquiera podría falsificar X-Forwarded-For y rotar de "IP" en cada
	// request para evadir el rate limit. Restringimos la confianza a los
	// rangos privados RFC1918 — el salto real del balanceador de Render
	// hacia este contenedor, no alcanzable directamente desde internet.
	if err := r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}); err != nil {
		log.Fatalf("setting trusted proxies: %v", err)
	}
	r.Use(middleware.MaxBodyBytes(1 << 20)) // 1 MiB por petición
	r.GET("/health", healthHandler.Check)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authGroup := r.Group("/auth")
	authGroup.Use(middleware.RateLimit(5.0/60, 5)) // 5 peticiones/min por IP, ráfaga de 5
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
		protected.POST("/appointments", appointmentHandler.Create)
		protected.PATCH("/appointments/:id/cancel", appointmentHandler.Cancel)

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
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("citas-api listening on %s (env=%s)", addr, cfg.AppEnv)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
