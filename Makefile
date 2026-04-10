APP_NAME=taskflow-api
COMPOSE=docker compose
COMPOSE_DEV=docker compose -f docker-compose.yml -f docker-compose.dev.yml

.PHONY: help deps format test run dev air-install build swagger docker-up docker-down docker-logs docker-ps docker-dev-up docker-dev-down docker-dev-logs docker-dev-ps vuln-scan ensure-env

help:
	@echo "Available targets:"
	@echo "  deps       - download and tidy Go dependencies"
	@echo "  ensure-env - create .env from .env.example if missing"
	@echo "  format     - format all Go files"
	@echo "  test       - run unit tests"
	@echo "  run        - run API locally (needs DATABASE_URL, JWT_SECRET, etc.)"
	@echo "  dev        - run API with Air live reload"
	@echo "  air-install- install Air CLI locally"
	@echo "  build      - build API binary"
	@echo "  swagger    - generate OpenAPI docs in ./docs"
	@echo "  docker-up  - start full stack with Docker (detached)"
	@echo "  docker-down- stop Docker stack"
	@echo "  docker-ps  - show Docker stack status"
	@echo "  docker-logs- follow Docker logs"
	@echo "  docker-dev-up   - start stack in development mode with Air"
	@echo "  docker-dev-down - stop development Docker stack"
	@echo "  docker-dev-ps   - show development Docker stack status"
	@echo "  docker-dev-logs - follow development Docker logs"
	@echo "  vuln-scan  - run govulncheck and gosec"

ensure-env:
	@if [ ! -f .env ]; then cp .env.example .env; fi

deps:
	go mod tidy

format:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

test:
	go test ./...

run:
	go run ./cmd/server

dev:
	go run github.com/cosmtrek/air@v1.49.0 -c .air.toml

air-install:
	go install github.com/cosmtrek/air@v1.49.0

build:
	go build -o bin/$(APP_NAME) ./cmd/server

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/server/main.go -o docs --parseDependency --parseInternal

docker-up: ensure-env
	$(COMPOSE) up --build -d
	$(COMPOSE) ps

docker-down:
	$(COMPOSE) down

docker-logs:
	$(COMPOSE) logs -f

docker-ps:
	$(COMPOSE) ps

docker-dev-up: ensure-env
	$(COMPOSE_DEV) up --build -d
	$(COMPOSE_DEV) ps

docker-dev-down:
	$(COMPOSE_DEV) down

docker-dev-logs:
	$(COMPOSE_DEV) logs -f

docker-dev-ps:
	$(COMPOSE_DEV) ps

vuln-scan:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	go run github.com/securego/gosec/v2/cmd/gosec@latest ./...
