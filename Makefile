APP_NAME=taskflow-api

.PHONY: help deps test run build swagger docker-up docker-down docker-logs docker-ps vuln-scan ensure-env

help:
	@echo "Available targets:"
	@echo "  deps       - download and tidy Go dependencies"
	@echo "  test       - run unit tests"
	@echo "  run        - run API locally (needs DATABASE_URL, JWT_SECRET, etc.)"
	@echo "  build      - build API binary"
	@echo "  swagger    - generate OpenAPI docs in ./docs"
	@echo "  docker-up  - start full stack with Docker (detached)"
	@echo "  docker-down- stop Docker stack"
	@echo "  docker-ps  - show Docker stack status"
	@echo "  docker-logs- follow Docker logs"
	@echo "  vuln-scan  - run govulncheck and gosec"

deps:
	go mod tidy

test:
	go test ./...

run:
	go run ./cmd/server

build:
	go build -o bin/$(APP_NAME) ./cmd/server

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/server/main.go -o docs --parseDependency --parseInternal

docker-up:
	@if [ ! -f .env ]; then cp .env.example .env; fi
	docker compose up --build -d
	docker compose ps

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-ps:
	docker compose ps

vuln-scan:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	go run github.com/securego/gosec/v2/cmd/gosec@latest ./...
