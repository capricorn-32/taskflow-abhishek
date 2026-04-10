package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"taskflow/backend/internal/repository"
)

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
	if err := decodeJSONBody(w, r, &req); err != nil {
		WriteValidationError(w, map[string]string{"body": err.Error()})
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
		req.Priority = repository.TaskPriorityMedium.String()
	}
	priorityValue, err := repository.ParseTaskPriority(req.Priority)
	if err != nil {
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
		Priority:    priorityValue,
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
	if err := decodeJSONBody(w, r, &req); err != nil {
		WriteValidationError(w, map[string]string{"body": err.Error()})
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
		priorityValue, err := repository.ParseTaskPriority(*req.Priority)
		if err != nil {
			fields["priority"] = "must be one of low|medium|high"
		} else {
			updates["priority"] = priorityValue.String()
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
