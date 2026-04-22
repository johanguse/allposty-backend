.PHONY: dev run worker seed build build-worker test test-coverage lint tidy \
        docker-up docker-down setup-local

# ── Development (no Docker required) ────────────────────────────────────────
#
# Option A — cloud services (recommended, zero install):
#   Postgres: https://neon.tech   (free tier, copy the connection string)
#   Redis:    https://upstash.com (free tier, copy the REDIS_URL)
#   Set both in your .env file and run `make run` + `make worker`.
#
# Option B — local install:
#   macOS:  brew install postgresql redis && brew services start postgresql redis
#   Ubuntu: sudo apt install postgresql redis-server
#
# Option C — Docker (only if you prefer it):
#   make docker-up

# Live-reload API (requires: go install github.com/air-verse/air@latest)
dev:
	air -c .air.toml

# Run API directly
run:
	go run ./cmd/api

# Run background worker
worker:
	go run ./cmd/worker

# Seed development database (idempotent)
seed:
	go run ./cmd/seed

# ── Build ───────────────────────────────────────────────────────────────────

build:
	go build -o bin/api ./cmd/api

build-worker:
	go build -o bin/worker ./cmd/worker

# ── Tests ───────────────────────────────────────────────────────────────────

test:
	go test ./... -v -race

test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ── Quality ─────────────────────────────────────────────────────────────────

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

# ── Docker (optional) ───────────────────────────────────────────────────────

docker-up:
	docker compose up -d

docker-down:
	docker compose down
