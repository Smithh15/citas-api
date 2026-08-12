package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{DB: db, Redis: rdb}
}

func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	status := gin.H{"status": "ok"}
	code := http.StatusOK

	if err := h.DB.Ping(ctx); err != nil {
		status["status"] = "degraded"
		status["postgres"] = err.Error()
		code = http.StatusServiceUnavailable
	} else {
		status["postgres"] = "ok"
	}

	if err := h.Redis.Ping(ctx).Err(); err != nil {
		status["status"] = "degraded"
		status["redis"] = err.Error()
		code = http.StatusServiceUnavailable
	} else {
		status["redis"] = "ok"
	}

	c.JSON(code, status)
}
