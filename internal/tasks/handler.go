package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Smithh15/citas-api/internal/db/sqlc"
	"github.com/Smithh15/citas-api/internal/mailer"
)

type Handler struct {
	Queries     sqlc.Querier
	Mailer      mailer.Mailer
	HoldMinutes int
}

func NewHandler(q sqlc.Querier, m mailer.Mailer, holdMinutes int) *Handler {
	return &Handler{Queries: q, Mailer: m, HoldMinutes: holdMinutes}
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
	var p SendReminderPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		// Payload corrupto: reintentarlo nunca va a funcionar.
		return fmt.Errorf("unmarshal payload: %w: %w", err, asynq.SkipRetry)
	}

	var apptID pgtype.UUID
	if err := apptID.Scan(p.AppointmentID); err != nil {
		return fmt.Errorf("invalid appointment id: %w", asynq.SkipRetry)
	}

	appt, err := h.Queries.GetAppointmentForReminder(ctx, apptID)
	if err != nil {
		return fmt.Errorf("fetch appointment: %w", err) // sí reintentar: puede ser la base caída
	}

	// La cita pudo cancelarse entre que se agendó el recordatorio y ahora.
	if appt.Status != sqlc.AppointmentStatusPending && appt.Status != sqlc.AppointmentStatusConfirmed {
		log.Printf("skipping reminder for %s: status is %s", p.AppointmentID, appt.Status)
		return nil
	}

	subject := "Recordatorio de tu cita médica"
	body := fmt.Sprintf("Hola %s, te recordamos tu cita con %s el %s.",
		appt.PatientName, appt.DoctorName, appt.SlotStart.Time.Format("02/01/2006 15:04"))

	return h.Mailer.Send(appt.PatientEmail, subject, body)
}
