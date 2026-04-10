package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"taskflow/backend/internal/auth"
	"taskflow/backend/internal/httpapi"
)

func Auth(jwtManager *auth.JWTManager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get("Authorization")
			if raw == "" || !strings.HasPrefix(raw, "Bearer ") {
				httpapi.WriteUnauthorized(w)
				return
			}

			token := strings.TrimPrefix(raw, "Bearer ")
			claims, err := jwtManager.ParseToken(token)
			if err != nil {
				httpapi.WriteUnauthorized(w)
				return
			}

			uid, err := uuid.Parse(claims.UserID)
			if err != nil {
				httpapi.WriteUnauthorized(w)
				return
			}

			ctx := httpapi.WithUser(r.Context(), uid, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			logger.Info("http_request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.statusCode),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *statusResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
