package main

import (
	"log"

	"github.com/hibiken/asynq"

	"github.com/Smithh15/citas-api/internal/config"
	"github.com/Smithh15/citas-api/internal/tasks"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	scheduler := asynq.NewScheduler(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB},
		nil,
	)

	// cada 5 minutos
	if _, err := scheduler.Register("*/5 * * * *",
		asynq.NewTask(tasks.TypeReleaseExpiredAppointments, nil)); err != nil {
		log.Fatalf("registering scheduled task: %v", err)
	}

	log.Println("scheduler started")
	if err := scheduler.Run(); err != nil {
		log.Fatalf("scheduler error: %v", err)
	}
}
