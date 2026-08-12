package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/Smithh15/citas-api/internal/config"
	"github.com/Smithh15/citas-api/internal/db"
	"github.com/Smithh15/citas-api/internal/db/sqlc"
	"github.com/Smithh15/citas-api/internal/handlers"
)

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

	r := gin.Default()
	r.GET("/health", healthHandler.Check)

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
	}

	addr := ":" + cfg.AppPort
	log.Printf("citas-api listening on %s (env=%s)", addr, cfg.AppEnv)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
