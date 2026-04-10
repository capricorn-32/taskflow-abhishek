package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

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

// register godoc
// @Summary Register a new user
// @Description Creates a user account and returns a JWT token.
// @Description Validation rules: `name` and `email` are required, `password` must be at least 8 characters.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body RegisterRequest true "Registration payload"
// @Success 201 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/register [post]
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteValidationError(w, map[string]string{"body": "invalid json"})
		return
	}

	fields := map[string]string{}
	if strings.TrimSpace(req.Name) == "" {
		fields["name"] = "is required"
	}
	if strings.TrimSpace(req.Email) == "" {
		fields["email"] = "is required"
	}
	if len(req.Password) < 8 {
		fields["password"] = "must be at least 8 characters"
	}
	if len(fields) > 0 {
		WriteValidationError(w, fields)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		WriteInternal(w)
		return
	}

	user, err := h.store.CreateUser(r.Context(), strings.TrimSpace(req.Name), strings.ToLower(strings.TrimSpace(req.Email)), string(hash))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate email") {
			WriteValidationError(w, map[string]string{"email": "already in use"})
			return
		}
		WriteInternal(w)
		return
	}

	token, err := h.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		WriteInternal(w)
		return
	}

	WriteJSON(w, http.StatusCreated, AuthResponse{
		Token: token,
		User: UserResponse{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}

// login godoc
// @Summary Login user
// @Description Authenticates user credentials and returns a JWT token.
// @Description Validation rules: `email` and `password` are required.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body LoginRequest true "Login payload"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/login [post]
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteValidationError(w, map[string]string{"body": "invalid json"})
		return
	}

	fields := map[string]string{}
	if strings.TrimSpace(req.Email) == "" {
		fields["email"] = "is required"
	}
	if strings.TrimSpace(req.Password) == "" {
		fields["password"] = "is required"
	}
	if len(fields) > 0 {
		WriteValidationError(w, fields)
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteUnauthorized(w)
			return
		}
		WriteInternal(w)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		WriteUnauthorized(w)
		return
	}

	token, err := h.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		WriteInternal(w)
		return
	}

	WriteJSON(w, http.StatusOK, AuthResponse{
		Token: token,
		User: UserResponse{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}

// listProjects godoc
// @Summary List projects
// @Description Lists projects where the current user is an owner, assignee, or task creator.
// @Description Pagination: `page` and `limit` must be positive integers. `limit` is clamped to server max.
// @Tags Projects
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Page size" default(20)
// @Success 200 {object} ProjectListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects [get]
func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}
	page, limit := h.pagination(r)

	projects, err := h.store.ListProjectsForUser(r.Context(), userID, page, limit)
	if err != nil {
		WriteInternal(w)
		return
	}
	WriteJSON(w, http.StatusOK, ProjectListResponse{Projects: projects})
}

// createProject godoc
// @Summary Create project
// @Description Creates a project with current user as owner.
// @Description Validation rules: `name` is required and trimmed.
// @Tags Projects
// @Accept json
// @Produce json
// @Param payload body ProjectUpsertRequest true "Project payload"
// @Success 201 {object} repository.Project
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects [post]
func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}

	var req ProjectUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteValidationError(w, map[string]string{"body": "invalid json"})
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		WriteValidationError(w, map[string]string{"name": "is required"})
		return
	}

	project, err := h.store.CreateProject(r.Context(), userID, strings.TrimSpace(req.Name), strings.TrimSpace(req.Description))
	if err != nil {
		WriteInternal(w)
		return
	}
	WriteJSON(w, http.StatusCreated, project)
}

// getProject godoc
// @Summary Get project details
// @Description Returns project details and tasks for an accessible project.
// @Description Behavior: returns `404` for malformed/non-existent IDs and `403` when user has no access.
// @Tags Projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID" format(uuid)
// @Success 200 {object} ProjectDetailResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id} [get]
func (h *Handler) getProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteNotFound(w)
		return
	}

	project, err := h.store.GetProjectByID(r.Context(), projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteNotFound(w)
			return
		}
		WriteInternal(w)
		return
	}

	hasAccess, err := h.store.UserHasProjectAccess(r.Context(), projectID, userID)
	if err != nil {
		WriteInternal(w)
		return
	}
	if !hasAccess {
		WriteForbidden(w)
		return
	}

	tasks, err := h.store.ListTasksByProject(r.Context(), projectID, repository.TaskFilters{Page: 1, Limit: h.maxPageSize})
	if err != nil {
		WriteInternal(w)
		return
	}

	WriteJSON(w, http.StatusOK, ProjectDetailResponse{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		OwnerID:     project.OwnerID,
		CreatedAt:   project.CreatedAt,
		Tasks:       tasks,
	})
}

// updateProject godoc
// @Summary Update project
// @Description Updates project name and description.
// @Description Authorization: only project owner can update. `name` is required.
// @Tags Projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID" format(uuid)
// @Param payload body ProjectUpsertRequest true "Project payload"
// @Success 200 {object} repository.Project
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id} [patch]
func (h *Handler) updateProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteNotFound(w)
		return
	}
	project, err := h.store.GetProjectByID(r.Context(), projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteNotFound(w)
			return
		}
		WriteInternal(w)
		return
	}
	if project.OwnerID != userID {
		WriteForbidden(w)
		return
	}

	var req ProjectUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteValidationError(w, map[string]string{"body": "invalid json"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		WriteValidationError(w, map[string]string{"name": "is required"})
		return
	}

	updated, err := h.store.UpdateProject(r.Context(), projectID, strings.TrimSpace(req.Name), strings.TrimSpace(req.Description))
	if err != nil {
		WriteInternal(w)
		return
	}
	WriteJSON(w, http.StatusOK, updated)
}

// deleteProject godoc
// @Summary Delete project
// @Description Deletes a project and all tasks under it.
// @Description Authorization: only project owner can delete.
// @Tags Projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID" format(uuid)
// @Success 204 {string} string "No Content"
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id} [delete]
func (h *Handler) deleteProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteNotFound(w)
		return
	}
	project, err := h.store.GetProjectByID(r.Context(), projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteNotFound(w)
			return
		}
		WriteInternal(w)
		return
	}
	if project.OwnerID != userID {
		WriteForbidden(w)
		return
	}

	if err := h.store.DeleteProject(r.Context(), projectID); err != nil {
		WriteInternal(w)
		return
	}
	WriteNoContent(w)
}

// listTasks godoc
// @Summary List tasks in project
// @Description Lists tasks for an accessible project with optional status/assignee filters.
// @Description Validation: `status` must be one of `todo|in_progress|done`; `assignee` must be UUID when provided.
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path string true "Project ID" format(uuid)
// @Param status query string false "Task status" Enums(todo,in_progress,done)
// @Param assignee query string false "Assignee user ID" format(uuid)
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Page size" default(20)
// @Success 200 {object} TaskListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id}/tasks [get]
func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteNotFound(w)
		return
	}

	hasAccess, err := h.store.UserHasProjectAccess(r.Context(), projectID, userID)
	if err != nil {
		WriteInternal(w)
		return
	}
	if !hasAccess {
		WriteForbidden(w)
		return
	}

	status := r.URL.Query().Get("status")
	assignee := r.URL.Query().Get("assignee")
	if status != "" && !isValidStatus(status) {
		WriteValidationError(w, map[string]string{"status": "must be one of todo|in_progress|done"})
		return
	}
	if assignee != "" {
		if _, err := uuid.Parse(assignee); err != nil {
			WriteValidationError(w, map[string]string{"assignee": "must be a valid uuid"})
			return
		}
	}
	page, limit := h.pagination(r)

	tasks, err := h.store.ListTasksByProject(r.Context(), projectID, repository.TaskFilters{Status: status, Assignee: assignee, Page: page, Limit: limit})
	if err != nil {
		WriteInternal(w)
		return
	}
	WriteJSON(w, http.StatusOK, TaskListResponse{Tasks: tasks})
}

// createTask godoc
// @Summary Create task
// @Description Creates a task inside a project.
// @Description Defaults: `status=todo`, `priority=medium` if omitted.
// @Description Validation: `title` required; `status` in `todo|in_progress|done`; `priority` in `low|medium|high`; `due_date` in `YYYY-MM-DD`.
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path string true "Project ID" format(uuid)
// @Param payload body TaskCreateRequest true "Task payload"
// @Success 201 {object} repository.Task
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id}/tasks [post]
func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteNotFound(w)
		return
	}

	hasAccess, err := h.store.UserHasProjectAccess(r.Context(), projectID, userID)
	if err != nil {
		WriteInternal(w)
		return
	}
	if !hasAccess {
		WriteForbidden(w)
		return
	}

	var req TaskCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteValidationError(w, map[string]string{"body": "invalid json"})
		return
	}

	fields := map[string]string{}
	if strings.TrimSpace(req.Title) == "" {
		fields["title"] = "is required"
	}
	if req.Status == "" {
		req.Status = "todo"
	}
	if !isValidStatus(req.Status) {
		fields["status"] = "must be one of todo|in_progress|done"
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}
	if !isValidPriority(req.Priority) {
		fields["priority"] = "must be one of low|medium|high"
	}

	var assigneeID *uuid.UUID
	if strings.TrimSpace(req.AssigneeID) != "" {
		assignee, err := uuid.Parse(req.AssigneeID)
		if err != nil {
			fields["assignee_id"] = "must be a valid uuid"
		} else {
			assigneeID = &assignee
		}
	}

	dueDate, err := repository.ParseDatePointer(req.DueDate)
	if err != nil {
		fields["due_date"] = "must be in YYYY-MM-DD format"
	}

	if len(fields) > 0 {
		WriteValidationError(w, fields)
		return
	}

	task, err := h.store.CreateTask(r.Context(), repository.Task{
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Status:      req.Status,
		Priority:    req.Priority,
		ProjectID:   projectID,
		AssigneeID:  assigneeID,
		CreatorID:   userID,
		DueDate:     dueDate,
	})
	if err != nil {
		WriteInternal(w)
		return
	}
	WriteJSON(w, http.StatusCreated, task)
}

// updateTask godoc
// @Summary Update task
// @Description Partially updates task fields.
// @Description Authorization: only project owner or task creator can update.
// @Description Validation: if provided, `title` cannot be empty, `status` and `priority` must be valid enums, `due_date` must be `YYYY-MM-DD`.
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Param payload body TaskUpdateRequest true "Partial task payload"
// @Success 200 {object} repository.Task
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id} [patch]
func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteNotFound(w)
		return
	}

	task, err := h.store.GetTaskByID(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteNotFound(w)
			return
		}
		WriteInternal(w)
		return
	}
	project, err := h.store.GetProjectByID(r.Context(), task.ProjectID)
	if err != nil {
		WriteInternal(w)
		return
	}
	if project.OwnerID != userID && task.CreatorID != userID {
		WriteForbidden(w)
		return
	}

	var req TaskUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteValidationError(w, map[string]string{"body": "invalid json"})
		return
	}

	fields := map[string]string{}
	updates := map[string]any{}

	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			fields["title"] = "cannot be empty"
		} else {
			updates["title"] = strings.TrimSpace(*req.Title)
		}
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil {
		if !isValidStatus(*req.Status) {
			fields["status"] = "must be one of todo|in_progress|done"
		} else {
			updates["status"] = *req.Status
		}
	}
	if req.Priority != nil {
		if !isValidPriority(*req.Priority) {
			fields["priority"] = "must be one of low|medium|high"
		} else {
			updates["priority"] = *req.Priority
		}
	}
	if req.AssigneeID != nil {
		if strings.TrimSpace(*req.AssigneeID) == "" {
			updates["assignee_id"] = nil
		} else {
			assignee, err := uuid.Parse(*req.AssigneeID)
			if err != nil {
				fields["assignee_id"] = "must be a valid uuid"
			} else {
				updates["assignee_id"] = assignee
			}
		}
	}
	if req.DueDate != nil {
		if strings.TrimSpace(*req.DueDate) == "" {
			updates["due_date"] = nil
		} else {
			dueDate, err := time.Parse("2006-01-02", *req.DueDate)
			if err != nil {
				fields["due_date"] = "must be in YYYY-MM-DD format"
			} else {
				updates["due_date"] = dueDate
			}
		}
	}

	if len(fields) > 0 {
		WriteValidationError(w, fields)
		return
	}

	updated, err := h.store.UpdateTask(r.Context(), taskID, updates)
	if err != nil {
		WriteInternal(w)
		return
	}
	WriteJSON(w, http.StatusOK, updated)
}

// deleteTask godoc
// @Summary Delete task
// @Description Deletes a task.
// @Description Authorization: only project owner or task creator can delete.
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID" format(uuid)
// @Success 204 {string} string "No Content"
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id} [delete]
func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteNotFound(w)
		return
	}

	task, err := h.store.GetTaskByID(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteNotFound(w)
			return
		}
		WriteInternal(w)
		return
	}
	project, err := h.store.GetProjectByID(r.Context(), task.ProjectID)
	if err != nil {
		WriteInternal(w)
		return
	}

	if project.OwnerID != userID && task.CreatorID != userID {
		WriteForbidden(w)
		return
	}

	if err := h.store.DeleteTask(r.Context(), taskID); err != nil {
		WriteInternal(w)
		return
	}
	WriteNoContent(w)
}

// projectStats godoc
// @Summary Project task statistics
// @Description Returns aggregate counts by task status and assignee.
// @Tags Projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID" format(uuid)
// @Success 200 {object} ProjectStatsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id}/stats [get]
func (h *Handler) projectStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteNotFound(w)
		return
	}
	hasAccess, err := h.store.UserHasProjectAccess(r.Context(), projectID, userID)
	if err != nil {
		WriteInternal(w)
		return
	}
	if !hasAccess {
		WriteForbidden(w)
		return
	}

	stats, err := h.store.ProjectStats(r.Context(), projectID)
	if err != nil {
		WriteInternal(w)
		return
	}
	WriteJSON(w, http.StatusOK, ProjectStatsResponse{
		ByStatus:   extractCounterMap(stats, "by_status"),
		ByAssignee: extractCounterMap(stats, "by_assignee"),
	})
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

func isValidPriority(v string) bool {
	switch v {
	case "low", "medium", "high":
		return true
	default:
		return false
	}
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
