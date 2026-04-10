package httpapi

import (
	"time"

	"github.com/google/uuid"

	"taskflow/backend/internal/repository"
)

// RegisterRequest is the payload for creating a new user account.
type RegisterRequest struct {
	Name     string `json:"name" example:"Jane Doe"`
	Email    string `json:"email" example:"jane@example.com"`
	Password string `json:"password" example:"password123"`
}

// LoginRequest is the payload for creating a JWT access token.
type LoginRequest struct {
	Email    string `json:"email" example:"jane@example.com"`
	Password string `json:"password" example:"password123"`
}

// UserResponse is the safe user profile returned in auth responses.
type UserResponse struct {
	ID    uuid.UUID `json:"id" example:"11111111-1111-1111-1111-111111111111"`
	Name  string    `json:"name" example:"Jane Doe"`
	Email string    `json:"email" example:"jane@example.com"`
}

// AuthResponse returns JWT and user profile after register/login.
type AuthResponse struct {
	Token string       `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User  UserResponse `json:"user"`
}

// HealthResponse is the liveness response for service health checks.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// ProjectUpsertRequest is used to create and update projects.
type ProjectUpsertRequest struct {
	Name        string `json:"name" example:"Website Redesign"`
	Description string `json:"description" example:"Q2 redesign project"`
}

// ProjectListResponse contains paginated project list results.
type ProjectListResponse struct {
	Projects []repository.Project `json:"projects"`
}

// TaskListResponse contains paginated task list results.
type TaskListResponse struct {
	Tasks []repository.Task `json:"tasks"`
}

// ProjectDetailResponse returns project details with nested tasks.
type ProjectDetailResponse struct {
	ID          uuid.UUID         `json:"id" example:"22222222-2222-2222-2222-222222222222"`
	Name        string            `json:"name" example:"Website Redesign"`
	Description string            `json:"description,omitempty" example:"Q2 redesign project"`
	OwnerID     uuid.UUID         `json:"owner_id" example:"11111111-1111-1111-1111-111111111111"`
	CreatedAt   time.Time         `json:"created_at"`
	Tasks       []repository.Task `json:"tasks"`
}

// TaskCreateRequest is the payload to create a task inside a project.
type TaskCreateRequest struct {
	Title       string `json:"title" example:"Design home page"`
	Description string `json:"description" example:"Build first draft with mobile variant"`
	Status      string `json:"status" example:"todo" enums:"todo,in_progress,done"`
	Priority    string `json:"priority" example:"high" enums:"low,medium,high"`
	AssigneeID  string `json:"assignee_id" example:"11111111-1111-1111-1111-111111111111"`
	DueDate     string `json:"due_date" example:"2026-04-30"`
}

// TaskUpdateRequest is the partial payload used to update a task.
type TaskUpdateRequest struct {
	Title       *string `json:"title" example:"Design dashboard"`
	Description *string `json:"description" example:"Update mockups and spacing"`
	Status      *string `json:"status" example:"in_progress" enums:"todo,in_progress,done"`
	Priority    *string `json:"priority" example:"medium" enums:"low,medium,high"`
	AssigneeID  *string `json:"assignee_id" example:"11111111-1111-1111-1111-111111111111"`
	DueDate     *string `json:"due_date" example:"2026-05-15"`
}

// ProjectStatsResponse summarizes task counts by status and assignee.
type ProjectStatsResponse struct {
	ByStatus   map[string]int `json:"by_status"`
	ByAssignee map[string]int `json:"by_assignee"`
}
