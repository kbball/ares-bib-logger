-include .env
export

DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)
MIGRATIONS_DIR=internal/adapter/repository/migrations

.DEFAULT_GOAL := help

.PHONY: help dev db-up db-down build build-backend build-frontend \
        test test-backend test-frontend \
        coverage coverage-backend coverage-frontend \
        lint lint-backend lint-frontend \
        fmt fmt-backend fmt-frontend \
        migrate-up migrate-down migrate-create migrate-status \
        install install-tools docker-build

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Dev"
	@echo "  db-up               Start Postgres in Docker (detached)"
	@echo "  db-down             Stop Postgres"
	@echo "  dev                 Start backend + frontend for local development"
	@echo "  build               Build backend binary and frontend bundle"
	@echo ""
	@echo "Testing"
	@echo "  test                Run all tests"
	@echo "  coverage            Run all tests with coverage reports"
	@echo ""
	@echo "Quality"
	@echo "  lint                Lint backend and frontend"
	@echo "  fmt                 Format backend and frontend"
	@echo ""
	@echo "Migrations"
	@echo "  migrate-up          Apply all pending migrations"
	@echo "  migrate-down        Roll back the last migration"
	@echo "  migrate-create      Scaffold a new migration  (NAME=<description>)"
	@echo "  migrate-status      Show current migration version"
	@echo ""
	@echo "Tooling"
	@echo "  install             Install all tools and frontend deps"
	@echo "  install-tools       Install golangci-lint and migrate CLI"
	@echo "  docker-build        Build the Docker image"

# ── Dev ──────────────────────────────────────────────────────────────────────

db-up:
	docker-compose up -d db mosquitto

db-down:
	docker-compose down

dev: db-up
	@trap 'kill 0' SIGINT; \
	(cd backend && go run ./cmd/server) & \
	(cd frontend && npm run dev) & \
	wait

# ── Build ─────────────────────────────────────────────────────────────────────

build: build-backend build-frontend

build-backend:
	cd backend && go build -o bin/server ./cmd/server

build-frontend:
	cd frontend && npm run build

# ── Test ──────────────────────────────────────────────────────────────────────

test: test-backend test-frontend

test-backend:
	cd backend && go test ./...

# Runs repository integration tests against the local DB (requires make db-up first).
test-integration:
	cd backend && DB_TEST_DSN="host=$(DB_HOST) port=$(DB_PORT) dbname=$(DB_NAME) user=$(DB_USER) password=$(DB_PASSWORD) sslmode=$(DB_SSL_MODE)" go test ./internal/adapter/repository/... -v

test-frontend:
	cd frontend && npm test

# ── Coverage ──────────────────────────────────────────────────────────────────

coverage: coverage-backend coverage-frontend

coverage-backend:
	cd backend && go test -coverprofile=coverage.out ./...
	cd backend && go tool cover -html=coverage.out -o coverage.html
	@echo "Backend coverage report: backend/coverage.html"

coverage-frontend:
	cd frontend && npm run coverage

# ── Quality ───────────────────────────────────────────────────────────────────

lint: lint-backend lint-frontend

lint-backend:
	cd backend && golangci-lint run ./...

lint-frontend:
	cd frontend && npm run lint

fmt: fmt-backend fmt-frontend

fmt-backend:
	cd backend && gofmt -w .

fmt-frontend:
	cd frontend && npm run format

# ── Migrations ────────────────────────────────────────────────────────────────

migrate-up:
	cd backend && migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

migrate-down:
	cd backend && migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

migrate-create:
	@[ "$(NAME)" ] || (echo "Error: NAME is required — usage: make migrate-create NAME=<description>"; exit 1)
	cd backend && migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)

migrate-status:
	cd backend && migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" version

# ── Tooling ───────────────────────────────────────────────────────────────────

install: install-tools
	cd frontend && npm install

install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

docker-build:
	docker build -t ares-bib-logger .
