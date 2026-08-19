package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
