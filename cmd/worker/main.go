package main

import (
	"context"
	"log"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Smithh15/citas-api/internal/config"
	"github.com/Smithh15/citas-api/internal/db/sqlc"
	"github.com/Smithh15/citas-api/internal/tasks"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to postgres: %v", err)
	}
	defer pool.Close()
	queries := sqlc.New(pool)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB},
		asynq.Config{Concurrency: 5},
	)

	mux := asynq.NewServeMux()
	handler := tasks.NewHandler(queries)
	mux.HandleFunc(tasks.TypeReleaseExpiredAppointments, handler.HandleReleaseExpired)
	mux.HandleFunc(tasks.TypeSendAppointmentReminder, handler.HandleSendReminder)

	log.Println("worker started")
	if err := srv.Run(mux); err != nil {
		log.Fatalf("worker error: %v", err)
	}
}
