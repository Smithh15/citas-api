package tasks

import (
	"context"
	"fmt"
	"log"

	"github.com/hibiken/asynq"

	"github.com/Smithh15/citas-api/internal/db/sqlc"
)

type Handler struct {
	Queries     sqlc.Querier
	HoldMinutes int
}

func NewHandler(q sqlc.Querier, holdMinutes int) *Handler {
	return &Handler{Queries: q, HoldMinutes: holdMinutes}
}

func (h *Handler) HandleReleaseExpired(ctx context.Context, t *asynq.Task) error {
	released, err := h.Queries.ReleaseExpiredPendingAppointments(ctx, int32(h.HoldMinutes))
	if err != nil {
		// Devolver el error hace que asynq reintente la tarea con backoff
		// exponencial en vez de perder el ciclo de limpieza silenciosamente.
		// Es seguro porque el job es idempotente: si ya no hay pending vencidos,
		// la siguiente corrida simplemente no encuentra nada que cancelar.
		return fmt.Errorf("release expired appointments: %w", err)
	}

	if len(released) > 0 {
		log.Printf("released %d expired pending appointments", len(released))
	}
	return nil
}

func (h *Handler) HandleSendReminder(ctx context.Context, t *asynq.Task) error {
	log.Println("send reminder: not implemented yet")
	return nil // Día 15
}
