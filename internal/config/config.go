package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	JWTSecret       string
	JWTIssuer       string
	JWTDuration     time.Duration
	LogLevel        slog.Level
	AutoMigrate     bool
	MigrationsPath  string
	DefaultPageSize int
	MaxPageSize     int
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		JWTIssuer:       getEnv("JWT_ISSUER", "taskflow"),
		JWTDuration:     24 * time.Hour,
		LogLevel:        parseLevel(getEnv("LOG_LEVEL", "info")),
		AutoMigrate:     getEnv("AUTO_MIGRATE", "true") == "true",
		MigrationsPath:  getEnv("MIGRATIONS_PATH", "file://migrations"),
		DefaultPageSize: getEnvInt("DEFAULT_PAGE_SIZE", 20),
		MaxPageSize:     getEnvInt("MAX_PAGE_SIZE", 100),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func parseLevel(v string) slog.Level {
	switch v {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
