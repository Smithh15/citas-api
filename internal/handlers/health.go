package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

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

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	status := map[string]string{"status": "ok"}
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(status)
}
