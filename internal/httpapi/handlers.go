package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"taskflow/backend/internal/auth"
	"taskflow/backend/internal/repository"
)

type Handler struct {
	store           *repository.Store
	jwt             *auth.JWTManager
	defaultPageSize int
	maxPageSize     int
}

func NewHandler(store *repository.Store, jwt *auth.JWTManager, defaultPageSize, maxPageSize int) *Handler {
	return &Handler{store: store, jwt: jwt, defaultPageSize: defaultPageSize, maxPageSize: maxPageSize}
}

// Health godoc
// @Summary Health check
// @Description Returns service liveness status.
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func (h *Handler) RegisterAuthRoutes(r chi.Router) {
	r.Post("/register", h.register)
	r.Post("/login", h.login)
}

func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	r.Get("/projects", h.listProjects)
	r.Post("/projects", h.createProject)
	r.Get("/projects/{id}", h.getProject)
	r.Patch("/projects/{id}", h.updateProject)
	r.Delete("/projects/{id}", h.deleteProject)
	r.Get("/projects/{id}/tasks", h.listTasks)
	r.Post("/projects/{id}/tasks", h.createTask)
	r.Get("/projects/{id}/stats", h.projectStats)
	r.Patch("/tasks/{id}", h.updateTask)
	r.Delete("/tasks/{id}", h.deleteTask)
}

func (h *Handler) pagination(r *http.Request) (int, int) {
	page := 1
	limit := h.defaultPageSize

	if rawPage := r.URL.Query().Get("page"); rawPage != "" {
		if v, err := strconv.Atoi(rawPage); err == nil && v > 0 {
			page = v
		}
	}
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if v, err := strconv.Atoi(rawLimit); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > h.maxPageSize {
		limit = h.maxPageSize
	}
	return page, limit
}

func isValidStatus(v string) bool {
	switch v {
	case "todo", "in_progress", "done":
		return true
	default:
		return false
	}
}

func isValidPriority(v repository.TaskPriority) bool {
	return v.IsValid()
}

func extractCounterMap(src map[string]any, key string) map[string]int {
	out := map[string]int{}
	v, ok := src[key]
	if !ok {
		return out
	}

	raw, ok := v.(map[string]int)
	if ok {
		return raw
	}

	m, ok := v.(map[string]any)
	if !ok {
		return out
	}

	for k, val := range m {
		switch n := val.(type) {
		case int:
			out[k] = n
		case int64:
			out[k] = int(n)
		case float64:
			out[k] = int(n)
		}
	}
	return out
}
