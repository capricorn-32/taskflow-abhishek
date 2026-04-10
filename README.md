# TaskFlow Backend (Go)

TaskFlow is a backend-first task management system built for the engineering take-home assignment. It provides authenticated project and task management APIs with PostgreSQL persistence, SQL migrations, seed data, and Docker-based local setup.

## 1. Overview

This project implements the **Backend Engineer** track requirements.

Features implemented:
- User registration and login with JWT auth
- Project CRUD with ownership authorization
- Task CRUD with status and assignee filters
- Task stats endpoint per project (bonus)
- Pagination support on list endpoints (bonus)
- Structured JSON error responses and status-code semantics
- PostgreSQL migrations and seeded test data
- Dockerized setup with one-command startup

Tech stack:
- Go 1.23
- Chi router
- PostgreSQL 16
- pgx / pgxpool
- golang-migrate
- bcrypt (cost 12)
- JWT (24h expiry, includes user_id and email claims)
- slog structured logging

## 2. Architecture Decisions

### Project structure
- cmd/server: app entrypoint and graceful shutdown
- internal/app: dependency wiring and router composition
- internal/httpapi: handlers, request validation, response contracts
- internal/httpapi/middleware: auth and request logging middleware
- internal/repository: SQL data access and query logic
- internal/auth: JWT creation and parsing
- internal/db: DB connection and migration runner
- migrations: versioned SQL up/down files
- scripts/seed.sql: deterministic seed data

### Key design choices
- Plain SQL over ORM: explicit control over schema, queries, and performance.
- Repository layer: keeps handlers focused on HTTP concerns.
- Strict auth boundaries: 401 for missing/invalid auth, 403 for forbidden actions.
- Migration-at-startup: predictable schema on container boot.

### Tradeoffs and omissions
- Backend-only submission (no React frontend).
- Unit tests added; no full end-to-end integration test harness yet.
- Refresh tokens and role-based memberships are intentionally out of scope.

## 3. Running Locally

Prerequisites:
- Docker + Docker Compose

### Option A: With Makefile (recommended)

```bash
git clone https://github.com/capricorn-32/taskflow-abhishek.git
cd taskflow-abhishek
make docker-up
```

The `docker-up` target auto-creates `.env` from `.env.example` if missing.

Service URLs:
- API: http://localhost:8080
- Swagger UI: http://localhost:8080/swagger/index.html
- Postgres: localhost:5432

Verify health:

```bash
curl http://localhost:8080/health
```

Stop services:

```bash
make docker-down
```

### Option B: Raw Docker Compose

```bash
git clone https://github.com/capricorn-32/taskflow-abhishek.git
cd taskflow-abhishek
cp .env.example .env
docker compose up --build -d
```

## 4. Running Migrations

Migrations run automatically on API startup using:
- `AUTO_MIGRATE=true`
- `MIGRATIONS_PATH=file://migrations`

Migration files:
- `migrations/000001_init.up.sql`
- `migrations/000001_init.down.sql`

If needed, you can reset and rerun with:

```bash
make docker-down
docker volume rm greening-india-assingment_pgdata || true
make docker-up
```

## 5. Test Credentials

Seed user credentials:
- Email: test@example.com
- Password: password123

Seeded project ID:
- 22222222-2222-2222-2222-222222222222

## 6. API Reference

Base URL:
- http://localhost:8080

Auth header for protected endpoints:
- `Authorization: Bearer <token>`

### Auth
- POST `/auth/register`
- POST `/auth/login`

### Projects
- GET `/projects` (supports `?page=&limit=`)
- POST `/projects`
- GET `/projects/:id`
- PATCH `/projects/:id` (owner only)
- DELETE `/projects/:id` (owner only)
- GET `/projects/:id/stats` (bonus)

### Tasks
- GET `/projects/:id/tasks?status=todo&assignee=<uuid>&page=1&limit=20`
- POST `/projects/:id/tasks`
- PATCH `/tasks/:id`
- DELETE `/tasks/:id` (project owner or task creator)

### Error contract
- `400`: `{ "error": "validation failed", "fields": { ... } }`
- `401`: `{ "error": "unauthorized" }`
- `403`: `{ "error": "forbidden" }`
- `404`: `{ "error": "not found" }`

### OpenAPI / Swagger

Swagger docs are generated with Swaggo and committed in `docs/`.

Regenerate docs after changing handler annotations:

```bash
make swagger
```

Swagger UI is served by the API at:

```text
http://localhost:8080/swagger/index.html
```

For protected endpoints in Swagger UI:
- Click `Authorize`
- Enter `Authorization` value as `Bearer <JWT_TOKEN>`
- Do not paste raw token without the `Bearer ` prefix

### Request examples
Login:

```bash
curl -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"password123"}'
```

Create project:

```bash
curl -X POST http://localhost:8080/projects \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"name":"New Project","description":"Optional"}'
```

Create task:

```bash
curl -X POST http://localhost:8080/projects/22222222-2222-2222-2222-222222222222/tasks \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"title":"Design homepage","priority":"high","status":"todo","due_date":"2026-04-30"}'
```

Postman collection:
- `postman/taskflow.postman_collection.json`

## 7. What I'd Do With More Time

- Add integration tests against ephemeral Postgres (auth + authorization flows).
- Add OpenAPI spec generation and contract validation.
- Add refresh-token flow and token rotation.
- Add rate limiting and brute-force protection for login.
- Add CI pipeline for lint, test, vulnerability scan, and container build.

## Useful Make Targets

```bash
make help
make deps
make test
make run
make build
make docker-up
make docker-ps
make docker-logs
make docker-down
make vuln-scan
```
