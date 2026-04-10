package repository

import (
	"time"

	"github.com/google/uuid"
)

// User represents an authenticated account in the system.
type User struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string
	CreatedAt time.Time `json:"created_at"`
}

// Project represents a work container that groups related tasks.
type Project struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	OwnerID     uuid.UUID `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// Task represents a unit of work belonging to a project.
type Task struct {
	ID          uuid.UUID    `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Status      string       `json:"status"`
	Priority    TaskPriority `json:"priority"`
	ProjectID   uuid.UUID    `json:"project_id"`
	AssigneeID  *uuid.UUID   `json:"assignee_id,omitempty"`
	CreatorID   uuid.UUID    `json:"creator_id"`
	DueDate     *time.Time   `json:"due_date,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type TaskFilters struct {
	Status   string
	Assignee string
	Page     int
	Limit    int
}
