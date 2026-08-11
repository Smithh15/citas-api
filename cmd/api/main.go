package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Smithh15/citas-api/internal/config"
	"github.com/Smithh15/citas-api/internal/db"
	"github.com/Smithh15/citas-api/internal/handlers"
	"github.com/Smithh15/citas-api/internal/middleware"
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

	healthHandler := handlers.NewHealthHandler(pgPool, redisClient)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/health", healthHandler.Check)

	addr := ":" + cfg.AppPort
	log.Printf("citas-api listening on %s (env=%s)", addr, cfg.AppEnv)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
