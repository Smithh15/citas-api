package tasks

import (
	"context"
	"log"

	"github.com/hibiken/asynq"

	"github.com/Smithh15/citas-api/internal/db/sqlc"
)

type Handler struct {
	Queries sqlc.Querier
}

func NewHandler(q sqlc.Querier) *Handler {
	return &Handler{Queries: q}
}

func (h *Handler) HandleReleaseExpired(ctx context.Context, t *asynq.Task) error {
	log.Println("release expired appointments: not implemented yet")
	return nil // Día 14
}

func (h *Handler) HandleSendReminder(ctx context.Context, t *asynq.Task) error {
	log.Println("send reminder: not implemented yet")
	return nil // Día 15
}
