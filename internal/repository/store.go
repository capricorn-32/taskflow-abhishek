package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) CreateUser(ctx context.Context, name, email, passwordHash string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		INSERT INTO users (name, email, password)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, password, created_at
	`, name, email, passwordHash).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, fmt.Errorf("duplicate email")
		}
		return User{}, err
	}
	return u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		SELECT id, name, email, password, created_at FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		SELECT id, name, email, password, created_at FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (s *Store) CreateProject(ctx context.Context, ownerID uuid.UUID, name, description string) (Project, error) {
	var p Project
	err := s.db.QueryRow(ctx, `
		INSERT INTO projects (owner_id, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, owner_id, created_at
	`, ownerID, name, description).Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt)
	if err != nil {
		return Project{}, err
	}
	return p, nil
}

func (s *Store) ListProjectsForUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]Project, error) {
	offset := (page - 1) * limit
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT p.id, p.name, p.description, p.owner_id, p.created_at
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.owner_id = $1 OR t.assignee_id = $1 OR t.creator_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]Project, 0)
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *Store) GetProjectByID(ctx context.Context, id uuid.UUID) (Project, error) {
	var p Project
	err := s.db.QueryRow(ctx, `
		SELECT id, name, description, owner_id, created_at
		FROM projects
		WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, err
	}
	return p, nil
}

func (s *Store) UserHasProjectAccess(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM projects p
			LEFT JOIN tasks t ON t.project_id = p.id
			WHERE p.id = $1
			AND (p.owner_id = $2 OR t.assignee_id = $2 OR t.creator_id = $2)
		)
	`, projectID, userID).Scan(&exists)
	return exists, err
}

func (s *Store) UpdateProject(ctx context.Context, id uuid.UUID, name, description string) (Project, error) {
	var p Project
	err := s.db.QueryRow(ctx, `
		UPDATE projects
		SET name = $2, description = $3
		WHERE id = $1
		RETURNING id, name, description, owner_id, created_at
	`, id, name, description).Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, err
	}
	return p, nil
}

func (s *Store) DeleteProject(ctx context.Context, id uuid.UUID) error {
	cmd, err := s.db.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListTasksByProject(ctx context.Context, projectID uuid.UUID, filters TaskFilters) ([]Task, error) {
	query := `
		SELECT id, title, description, status, priority, project_id, assignee_id, creator_id, due_date, created_at, updated_at
		FROM tasks
		WHERE project_id = $1
	`

	args := []any{projectID}
	argPos := 2

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filters.Status)
		argPos++
	}
	if filters.Assignee != "" {
		query += fmt.Sprintf(" AND assignee_id = $%d", argPos)
		args = append(args, filters.Assignee)
		argPos++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.ProjectID, &t.AssigneeID, &t.CreatorID, &t.DueDate, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	return tasks, rows.Err()
}

func (s *Store) CreateTask(ctx context.Context, input Task) (Task, error) {
	var t Task
	err := s.db.QueryRow(ctx, `
		INSERT INTO tasks (title, description, status, priority, project_id, assignee_id, creator_id, due_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, title, description, status, priority, project_id, assignee_id, creator_id, due_date, created_at, updated_at
	`, input.Title, input.Description, input.Status, input.Priority, input.ProjectID, input.AssigneeID, input.CreatorID, input.DueDate).Scan(
		&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.ProjectID, &t.AssigneeID, &t.CreatorID, &t.DueDate, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s *Store) GetTaskByID(ctx context.Context, id uuid.UUID) (Task, error) {
	var t Task
	err := s.db.QueryRow(ctx, `
		SELECT id, title, description, status, priority, project_id, assignee_id, creator_id, due_date, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`).Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.ProjectID, &t.AssigneeID, &t.CreatorID, &t.DueDate, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, err
	}
	return t, nil
}

func (s *Store) UpdateTask(ctx context.Context, taskID uuid.UUID, fields map[string]any) (Task, error) {
	if len(fields) == 0 {
		return s.GetTaskByID(ctx, taskID)
	}

	allowedColumns := map[string]struct{}{
		"title":       {},
		"description": {},
		"status":      {},
		"priority":    {},
		"assignee_id": {},
		"due_date":    {},
	}

	setParts := make([]string, 0, len(fields)+1)
	args := make([]any, 0, len(fields)+1)
	idx := 1

	for key, value := range fields {
		if _, ok := allowedColumns[key]; !ok {
			return Task{}, fmt.Errorf("invalid update field: %s", key)
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, idx))
		args = append(args, value)
		idx++
	}
	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, taskID)

	query := fmt.Sprintf(`
		UPDATE tasks
		SET %s
		WHERE id = $%d
		RETURNING id, title, description, status, priority, project_id, assignee_id, creator_id, due_date, created_at, updated_at
	`, strings.Join(setParts, ", "), idx)

	var t Task
	err := s.db.QueryRow(ctx, query, args...).Scan(
		&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.ProjectID, &t.AssigneeID, &t.CreatorID, &t.DueDate, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, err
	}
	return t, nil
}

func (s *Store) DeleteTask(ctx context.Context, id uuid.UUID) error {
	cmd, err := s.db.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	statusCounts := map[string]int{"todo": 0, "in_progress": 0, "done": 0}
	rows, err := s.db.Query(ctx, `
		SELECT status, COUNT(*)
		FROM tasks
		WHERE project_id = $1
		GROUP BY status
	`, projectID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			rows.Close()
			return nil, err
		}
		statusCounts[status] = count
	}
	rows.Close()

	assigneeRows, err := s.db.Query(ctx, `
		SELECT COALESCE(assignee_id::text, 'unassigned') AS assignee, COUNT(*)
		FROM tasks
		WHERE project_id = $1
		GROUP BY assignee
	`, projectID)
	if err != nil {
		return nil, err
	}
	assigneeCounts := map[string]int{}
	for assigneeRows.Next() {
		var assignee string
		var count int
		if err := assigneeRows.Scan(&assignee, &count); err != nil {
			assigneeRows.Close()
			return nil, err
		}
		assigneeCounts[assignee] = count
	}
	assigneeRows.Close()

	return map[string]any{
		"by_status":   statusCounts,
		"by_assignee": assigneeCounts,
	}, nil
}

func ParseDatePointer(v string) (*time.Time, error) {
	if strings.TrimSpace(v) == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
