APP_NAME=taskflow-api

.PHONY: help deps test run build docker-up docker-down vuln-scan

help:
	@echo "Available targets:"
	@echo "  deps       - download and tidy Go dependencies"
	@echo "  test       - run unit tests"
	@echo "  run        - run API locally (needs DATABASE_URL, JWT_SECRET, etc.)"
	@echo "  build      - build API binary"
	@echo "  docker-up  - start full stack with Docker"
	@echo "  docker-down- stop Docker stack"
	@echo "  vuln-scan  - run govulncheck and gosec"

deps:
	go mod tidy

test:
	go test ./...

run:
	go run ./cmd/server

build:
	go build -o bin/$(APP_NAME) ./cmd/server

docker-up:
	docker compose up --build

docker-down:
	docker compose down

vuln-scan:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	go run github.com/securego/gosec/v2/cmd/gosec@latest ./...
