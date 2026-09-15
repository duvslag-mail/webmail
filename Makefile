-include .env
export

DB_URL ?= postgres://mailuser:secretpassword@localhost:5432/maildb?sslmode=disable
DB_USER ?= mailuser
DB_NAME ?= maildb

.PHONY: all dev generate migrate-up migrate-down build test clean

all: dev

dev:
	@echo "Starting postgres container..."
	docker compose up -d postgres
	@echo "Waiting for postgres to be ready..."
	@until docker exec webmail_db pg_isready -U $(DB_USER) -d $(DB_NAME) > /dev/null 2>&1; do sleep 1; done
	@echo "Running database migrations..."
	$(MAKE) migrate-up
	@echo "Generating templ and sqlc code..."
	$(MAKE) generate
	@echo "Starting air..."
	air

generate:
	templ generate
	sqlc generate

migrate-up:
	goose -dir db/migrations postgres "$(DB_URL)" up

migrate-down:
	goose -dir db/migrations postgres "$(DB_URL)" down

build: generate
	go build -ldflags="-s -w" -o bin/server ./cmd/server

test:
	go test -v -short ./...

clean:
	rm -rf bin/ tmp/ db/sqlc/* *_templ.go
