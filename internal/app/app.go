package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"taskflow/backend/internal/auth"
	"taskflow/backend/internal/config"
	"taskflow/backend/internal/db"
	"taskflow/backend/internal/httpapi"
	authmw "taskflow/backend/internal/httpapi/middleware"
	"taskflow/backend/internal/repository"
)

type App struct {
	cfg    config.Config
	logger *slog.Logger
	db     *pgxpool.Pool
	router http.Handler
}

func New(cfg config.Config, logger *slog.Logger) (*App, error) {
	ctx := context.Background()
	if cfg.AutoMigrate {
		if err := db.Migrate(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
			return nil, fmt.Errorf("migrate db: %w", err)
		}
		logger.Info("database migrations applied")
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	store := repository.New(pool)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTDuration)
	h := httpapi.NewHandler(store, jwtManager, cfg.DefaultPageSize, cfg.MaxPageSize)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(authmw.RequestLogger(logger))

	r.Get("/health", h.Health)
	r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

	r.Route("/auth", func(r chi.Router) {
		h.RegisterAuthRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(authmw.Auth(jwtManager))
		h.RegisterProtectedRoutes(r)
	})

	return &App{cfg: cfg, logger: logger, db: pool, router: r}, nil
}

func (a *App) Router() http.Handler {
	return a.router
}

func (a *App) Close() {
	a.db.Close()
}
