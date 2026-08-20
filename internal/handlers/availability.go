package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Smithh15/citas-api/internal/db/sqlc"
)

type AvailabilityHandler struct {
	Queries sqlc.Querier
}

type CreateAvailabilityRequest struct {
	DayOfWeek int16  `json:"day_of_week" binding:"min=0,max=6"`
	StartTime string `json:"start_time" binding:"required"` // formato "08:00"
	EndTime   string `json:"end_time" binding:"required"`
}

func parseTimeOfDay(s string) (pgtype.Time, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return pgtype.Time{}, err
	}
	micros := int64(t.Hour())*3600*1_000_000 + int64(t.Minute())*60*1_000_000
	return pgtype.Time{Microseconds: micros, Valid: true}, nil
}

func (h *AvailabilityHandler) Create(c *gin.Context) {
	var req CreateAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	start, err := parseTimeOfDay(req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time, use HH:MM"})
		return
	}
	end, err := parseTimeOfDay(req.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time, use HH:MM"})
		return
	}
	if end.Microseconds <= start.Microseconds {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_time must be after start_time"})
		return
	}

	var userID pgtype.UUID
	if err := userID.Scan(c.GetString("userID")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user"})
		return
	}

	profile, err := h.Queries.GetDoctorProfileByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "doctor profile not found"})
		return
	}

	avail, err := h.Queries.CreateAvailability(c.Request.Context(), sqlc.CreateAvailabilityParams{
		DoctorID:  profile.ID,
		DayOfWeek: req.DayOfWeek,
		StartTime: start,
		EndTime:   end,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create availability"})
		return
	}

	c.JSON(http.StatusCreated, avail)
}
