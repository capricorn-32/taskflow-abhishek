package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"taskflow/backend/internal/repository"
)

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
	if err := decodeJSONBody(w, r, &req); err != nil {
		WriteValidationError(w, map[string]string{"body": err.Error()})
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
	if err := decodeJSONBody(w, r, &req); err != nil {
		WriteValidationError(w, map[string]string{"body": err.Error()})
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
