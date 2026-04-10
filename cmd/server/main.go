package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "taskflow/backend/docs"

	"taskflow/backend/internal/app"
	"taskflow/backend/internal/config"
)

// @title TaskFlow API
// @version 1.0
// @description TaskFlow backend API for authentication, projects, and task management.
// @description
// @description Validation errors return `400` with `{ "error": "validation failed", "fields": { ... } }`.
// @description For protected endpoints, the Authorization header must be: `Bearer <JWT_TOKEN>`.
// @description In Swagger UI Authorize dialog, paste the value exactly as `Bearer <JWT_TOKEN>`.
// @BasePath /
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Required format: `Bearer <JWT_TOKEN>`.

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Error("failed to create app", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer application.Close()

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           application.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("starting server", slog.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("failed graceful shutdown", slog.String("error", err.Error()))
	}
}
