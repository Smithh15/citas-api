package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv               string
	AppPort              string
	DatabaseURL          string
	RedisAddr            string
	RedisPassword        string
	RedisDB              int
	JWTSecret            string
	MinCancellationHours int
	PendingHoldMinutes   int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	minCancelHours, err := strconv.Atoi(getEnv("MIN_CANCELLATION_HOURS", "24"))
	if err != nil {
		return nil, fmt.Errorf("invalid MIN_CANCELLATION_HOURS: %w", err)
	}

	pendingHoldMinutes, err := strconv.Atoi(getEnv("PENDING_HOLD_MINUTES", "15"))
	if err != nil {
		return nil, fmt.Errorf("invalid PENDING_HOLD_MINUTES: %w", err)
	}

	cfg := &Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		AppPort:              getEnv("APP_PORT", "8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:        os.Getenv("REDIS_PASSWORD"),
		RedisDB:              0,
		JWTSecret:            os.Getenv("JWT_SECRET"),
		MinCancellationHours: minCancelHours,
		PendingHoldMinutes:   pendingHoldMinutes,
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
