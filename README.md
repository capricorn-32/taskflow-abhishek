# TaskFlow Backend API

## 1. Project Overview
TaskFlow is a RESTful API backend built with Go (Chi router) and PostgreSQL for authentication, project management, and task management.
It provides JWT-protected endpoints, SQL migrations, and Docker-based local setup.

## 2. Prerequisites
- Go 1.23+
- PostgreSQL 16+
- Docker 24+ and Docker Compose v2+
- Make 4.x (optional, for helper commands)

## 3. Setup & Installation
```bash
git clone https://github.com/capricorn-32/taskflow-abhishek.git
cd taskflow-abhishek
go mod tidy
cp .env.example .env
docker compose up --build -d
```

Run server locally (without Docker API container):
```bash
export DATABASE_URL="postgres://taskflow:taskflow@localhost:5432/taskflow?sslmode=disable"
export JWT_SECRET="super-secret-change-me"
go run ./cmd/server
```

Default API port: `8080`.
Swagger API docs: `http://localhost:8080/swagger/index.html`.

Regenerate OpenAPI docs after API annotation changes:
```bash
make swagger
```

## 4. Environment Variables
| Variable | Description | Example Value |
|---|---|---|
| `HTTP_ADDR` | HTTP bind address for API server | `:8080` |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://taskflow:taskflow@localhost:5432/taskflow?sslmode=disable` |
| `JWT_SECRET` | Secret key for JWT signing | `super-secret-change-me` |
| `JWT_ISSUER` | JWT issuer claim | `taskflow` |
| `AUTO_MIGRATE` | Run migrations on startup | `true` |
| `MIGRATIONS_PATH` | Migration files location | `file://migrations` |
| `LOG_LEVEL` | Application log level | `info` |
| `DEFAULT_PAGE_SIZE` | Default pagination size | `20` |
| `MAX_PAGE_SIZE` | Maximum pagination size | `100` |
| `POSTGRES_DB` | PostgreSQL database name (Docker) | `taskflow` |
| `POSTGRES_USER` | PostgreSQL user (Docker) | `taskflow` |
| `POSTGRES_PASSWORD` | PostgreSQL password (Docker) | `taskflow` |

## 5. Database Setup
1. Start PostgreSQL and API with Docker:
```bash
docker compose up --build -d
```
2. Create DB manually if running PostgreSQL outside Docker:
```bash
createdb taskflow
```
3. Run migrations (automatic on server start):
```bash
AUTO_MIGRATE=true MIGRATIONS_PATH=file://migrations go run ./cmd/server
```

Special setup step: if `.env` does not exist, create it from `.env.example` before starting services.

## 6. API Endpoints
| Method | Endpoint | Description | Auth Required |
|---|---|---|---|
| `GET` | `/health` | Service health check | No |
| `POST` | `/auth/register` | Register a new user | No |
| `POST` | `/auth/login` | Login user and get JWT | No |
| `GET` | `/projects` | List user-accessible projects | Yes |
| `POST` | `/projects` | Create a project | Yes |
| `GET` | `/projects/{id}` | Get project details | Yes |
| `PATCH` | `/projects/{id}` | Update a project | Yes |
| `DELETE` | `/projects/{id}` | Delete a project | Yes |
| `GET` | `/projects/{id}/tasks` | List tasks in a project | Yes |
| `POST` | `/projects/{id}/tasks` | Create a task in a project | Yes |
| `GET` | `/projects/{id}/stats` | Get task statistics for a project | Yes |
| `PATCH` | `/tasks/{id}` | Update a task | Yes |
| `DELETE` | `/tasks/{id}` | Delete a task | Yes |

## 7. Project Structure
```text
.
├── cmd/server/main.go                 # Application entrypoint
├── internal/app/app.go                # Router and dependency wiring
├── internal/config/config.go          # Environment-based config loader
├── internal/httpapi/                  # HTTP handlers, DTOs, responses
│   └── middleware/                    # Auth and request logging middleware
├── internal/repository/               # SQL queries and data access layer
├── internal/db/                       # DB connection and migration runner
├── internal/auth/                     # JWT generation and validation
├── migrations/                        # SQL migration files
├── scripts/seed.sql                   # Seed data script
├── docs/                              # Swagger/OpenAPI generated docs
├── docker-compose.yml                 # Local API + PostgreSQL stack
└── Makefile                           # Common development commands
```

## 8. Request Flowchart
```mermaid
flowchart LR
    A[Client Request] --> B[Chi Router]
    B --> C[Middleware: Auth + Logger]
    C --> D[HTTP Handler]
    D --> E[Repository]
    E --> F[(PostgreSQL)]
    F --> E
    E --> D
    D --> G[JSON Response]
    G --> H[Client]
```

## 9. Backend Architecture
```mermaid
flowchart TB
    CL[Client Layer\nHTTP Requests]
    RL[Router Layer\nChi]
    ML[Middleware Layer\nJWT Auth, Logger]
    HL[Handler Layer\nBusiness Logic]
    RE[Repository Layer\nDB Queries]
    DB[(Database Layer\nPostgreSQL)]

    CL --> RL
    RL --> ML
    ML --> HL
    HL --> RE
    RE --> DB
    DB --> RE
    RE --> HL
    HL --> CL
```

## 10. Architecture Decisions
- Layered structure (`httpapi -> repository -> postgres`) keeps HTTP concerns separate from SQL and improves testability.
- Plain SQL with migration files was chosen over an ORM for explicit schema control and predictable queries.
- JWT auth is applied at router-group level so protected routes are enforced consistently.
- Automatic migrations on startup prioritize reviewer experience (`docker compose up` should be enough).
- Tradeoff: the project favors clarity and assignment scope over advanced patterns like domain events or CQRS.

## 11. What Is Missing / Next Improvements
- Add more end-to-end integration tests for auth, authorization, and task lifecycle paths.
- Add rate limiting and login brute-force protection for production hardening.
- Add request timeout middleware and per-route timeout tuning.
- Add CI pipeline for tests, linting, Swagger drift check, and vulnerability scan.
- Add refresh-token flow and token rotation strategy for long-lived sessions.

## 12. Tests
- Unit tests:
```bash
make test
```
- Integration tests (requires a running PostgreSQL and optional `TEST_DATABASE_URL`):
```bash
TEST_DATABASE_URL="postgres://taskflow:taskflow@localhost:5432/taskflow?sslmode=disable" go test ./internal/httpapi -run Integration
```
