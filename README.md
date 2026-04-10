# TaskFlow Backend (Go + PostgreSQL)

## 1. Overview
This submission implements the **backend track** of the TaskFlow take-home in Go.

It provides:
- JWT-based authentication (`/auth/register`, `/auth/login`)
- Project CRUD with ownership enforcement
- Task CRUD with status/assignee filters
- PostgreSQL schema via versioned SQL migrations (up + down)
- Dockerized local environment with one-command startup
- Seed data (test user, project, and 3 tasks with different statuses)

### Tech Stack
- Go 1.23
- `chi` router
- PostgreSQL 16
- `pgx` / `pgxpool`
- `golang-migrate` (migration runner in app startup)
- `bcrypt` for password hashing (cost 12)
- JWT (`HS256`, 24h expiry) with `user_id` + `email` claims
- Structured logging via `slog`

## 2. Architecture Decisions
### Why this structure
- `cmd/server`: app entrypoint + graceful shutdown
- `internal/config`: env-driven configuration
- `internal/db`: DB connection + migration orchestration
- `internal/repository`: SQL-centric data access layer
- `internal/httpapi`: handlers, request validation, error contracts
- `internal/httpapi/middleware`: auth and request logging
- `migrations`: explicit SQL schema history
- `scripts/seed.sql`: deterministic demo data

### Tradeoffs
- I intentionally used plain SQL (not an ORM) to keep behavior explicit and migration-safe.
- I used a repository layer rather than introducing service abstractions where they would be thin wrappers.
- Validation is handler-level and explicit for clarity over generic validation frameworks.

### Intentional omissions
- No React frontend in this repository (backend-role submission).
- No full integration test suite yet (Postman collection included for API verification).
- No refresh token flow (access-token-only per assignment scope).

## 3. Running Locally
Assumption: Docker + Docker Compose are installed.

```bash
git clone <your-repo-url>
cd Greening-India-Assingment
cp .env.example .env
docker compose up --build
```

### Makefile shortcuts
```bash
make deps
make test
make run        # run API locally (requires env vars)
make docker-up  # start full stack via Docker
make docker-down
make vuln-scan
```

Services:
- API: `http://localhost:8080`
- Postgres: `localhost:5432`

Health check:
```bash
curl http://localhost:8080/health
```

To stop:
```bash
docker compose down
```

To remove DB volume too:
```bash
docker compose down -v
```

## 4. Running Migrations
Migrations run automatically on API startup when:
- `AUTO_MIGRATE=true`
- `MIGRATIONS_PATH=file://migrations`

These are already set in `docker-compose.yml`.

Migration files:
- `migrations/000001_init.up.sql`
- `migrations/000001_init.down.sql`

## 5. Test Credentials
Seed data (`scripts/seed.sql`) creates:

- Email: `test@example.com`
- Password: `password123`

Seeded IDs:
- User ID: `11111111-1111-1111-1111-111111111111`
- Project ID: `22222222-2222-2222-2222-222222222222`

## 6. API Reference
Base URL: `http://localhost:8080`

All non-auth endpoints require:
`Authorization: Bearer <token>`

### Auth
- `POST /auth/register`
- `POST /auth/login`

Example request:
```json
{
  "name": "Jane Doe",
  "email": "jane@example.com",
  "password": "secret123"
}
```

Example login response:
```json
{
  "token": "<jwt>",
  "user": {
    "id": "uuid",
    "name": "Jane Doe",
    "email": "jane@example.com"
  }
}
```

### Projects
- `GET /projects` (supports `?page=&limit=`)
- `POST /projects`
- `GET /projects/:id` (returns project + tasks)
- `PATCH /projects/:id` (owner only)
- `DELETE /projects/:id` (owner only)
- `GET /projects/:id/stats` (bonus: counts by status and assignee)

### Tasks
- `GET /projects/:id/tasks?status=todo&assignee=<uuid>&page=1&limit=20`
- `POST /projects/:id/tasks`
- `PATCH /tasks/:id`
- `DELETE /tasks/:id` (project owner or task creator)

Example create task request:
```json
{
  "title": "Design homepage",
  "description": "Initial layout",
  "status": "todo",
  "priority": "high",
  "assignee_id": "11111111-1111-1111-1111-111111111111",
  "due_date": "2026-04-30"
}
```

### Error Contracts
- `400`: `{ "error": "validation failed", "fields": { ... } }`
- `401`: `{ "error": "unauthorized" }`
- `403`: `{ "error": "forbidden" }`
- `404`: `{ "error": "not found" }`

### Postman
A ready collection is included at:
- `postman/taskflow.postman_collection.json`

## 7. What I'd Do With More Time
- Add integration tests with a disposable Postgres container for auth + task authorization flows.
- Add rate limiting and login brute-force protections.
- Add OpenAPI spec generation and request/response contract tests.
- Add role-based project membership model beyond owner/task-based access.
- Add CI pipeline (lint, tests, image build, vulnerability scan).
