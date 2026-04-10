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

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
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

	WriteJSON(w, http.StatusCreated, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
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

	WriteJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	})
}

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
	WriteJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteUnauthorized(w)
		return
	}

	var req createProjectRequest
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

	WriteJSON(w, http.StatusOK, map[string]any{
		"id":          project.ID,
		"name":        project.Name,
		"description": project.Description,
		"owner_id":    project.OwnerID,
		"created_at":  project.CreatedAt,
		"tasks":       tasks,
	})
}

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

	var req createProjectRequest
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
	WriteJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	AssigneeID  string `json:"assignee_id"`
	DueDate     string `json:"due_date"`
}

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

	var req createTaskRequest
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

type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	AssigneeID  *string `json:"assignee_id"`
	DueDate     *string `json:"due_date"`
}

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

	var req updateTaskRequest
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
	WriteJSON(w, http.StatusOK, stats)
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
