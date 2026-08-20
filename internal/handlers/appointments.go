package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Smithh15/citas-api/internal/db/sqlc"
)

type AppointmentHandler struct {
	Queries sqlc.Querier
}

// GetAvailableSlots godoc
// @Summary      Consulta cupos disponibles de un doctor
// @Description  Calcula los horarios libres de un doctor en una fecha dada, cruzando su disponibilidad semanal contra las citas ya reservadas
// @Tags         appointments
// @Produce      json
// @Param        doctor_id query string true "ID del doctor (doctor_profiles.id)"
// @Param        date query string true "Fecha en formato YYYY-MM-DD"
// @Success      200 {object} map[string][]string
// @Failure      400 {object} map[string]string
// @Router       /appointments/available [get]
func (h *AppointmentHandler) GetAvailableSlots(c *gin.Context) {
	doctorIDParam := c.Query("doctor_id")
	dateParam := c.Query("date")

	var doctorID pgtype.UUID
	if err := doctorID.Scan(doctorIDParam); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid doctor_id"})
		return
	}

	parsedDate, err := time.Parse("2006-01-02", dateParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date, use YYYY-MM-DD"})
		return
	}
	targetDate := pgtype.Date{Time: parsedDate, Valid: true}

	slots, err := h.Queries.GetAvailableSlots(c.Request.Context(), sqlc.GetAvailableSlotsParams{
		DoctorID:   doctorID,
		TargetDate: targetDate,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch slots"})
		return
	}

	available := make([]string, 0, len(slots))
	for _, s := range slots {
		available = append(available, s.Time.Format(time.RFC3339))
	}

	c.JSON(http.StatusOK, gin.H{"available_slots": available})
}

type CreateAppointmentRequest struct {
	DoctorID  string `json:"doctor_id" binding:"required"`
	SlotStart string `json:"slot_start" binding:"required"` // RFC3339, ej: "2026-08-17T08:00:00-05:00"
}

// Create godoc
// @Summary      Reserva una cita
// @Description  Crea una cita para el usuario autenticado en el cupo indicado; si el cupo ya fue tomado, devuelve 409
// @Tags         appointments
// @Accept       json
// @Produce      json
// @Param        request body CreateAppointmentRequest true "Datos de la reserva"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Router       /appointments [post]
func (h *AppointmentHandler) Create(c *gin.Context) {
	var req CreateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var doctorID pgtype.UUID
	if err := doctorID.Scan(req.DoctorID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid doctor_id"})
		return
	}

	slotStart, err := time.Parse(time.RFC3339, req.SlotStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid slot_start, use RFC3339"})
		return
	}
	// TODO: usar default_slot_minutes del doctor en vez de un valor fijo
	slotEnd := slotStart.Add(30 * time.Minute)

	var patientID pgtype.UUID
	if err := patientID.Scan(c.GetString("userID")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user"})
		return
	}

	appt, err := h.Queries.CreateAppointment(c.Request.Context(), sqlc.CreateAppointmentParams{
		PatientID: patientID,
		DoctorID:  doctorID,
		SlotStart: pgtype.Timestamptz{Time: slotStart, Valid: true},
		SlotEnd:   pgtype.Timestamptz{Time: slotEnd, Valid: true},
	})
	if err != nil {
		// DO NOTHING en el ON CONFLICT hace que pgx no devuelva fila: esto es
		// exactamente lo que separa "cupo tomado" de un error real de la base.
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusConflict, gin.H{"error": "this slot is no longer available"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create appointment"})
		return
	}

	c.JSON(http.StatusCreated, appt)
}
