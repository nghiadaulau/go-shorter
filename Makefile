SHELL := /bin/bash
APP_NAME := shortlink
GO := go

.PHONY: dev run build test migrate seed help db-create db-reset migrate-sql

dev: ## Run api in dev mode
	$(GO) run ./cmd/shortlink-api --config ./config/config.yaml

run: ## Run redirector
	$(GO) run ./cmd/redirector --config ./config/config.yaml

build: ## Build binaries
	$(GO) build -o bin/shortlink-api ./cmd/shortlink-api
	$(GO) build -o bin/redirector ./cmd/redirector

test: ## Run tests
	$(GO) test ./... -count=1 -race -timeout=5m

migrate: ## Run migrations (requires golang-migrate installed)
	migrate -path migrations -database "${DATABASE_URL}" up

migrate-sql: ## Run SQL migrations via psql (no external CLI required)
	@if [ -z "${DATABASE_URL}" ]; then echo "DATABASE_URL not set"; exit 1; fi;
	psql "${DATABASE_URL}" -f migrations/0001_init.up.sql;
	psql "${DATABASE_URL}" -f migrations/0002_redirect_perf_indexes.up.sql;

seed: ## Seed admin
	$(GO) run ./cmd/shortlink-api seed --admin-email nhatnghiatyper@gmail.com --password Nghia1305 --tenant default --config ./config/config.yaml

# Create DB if missing (connect to 'postgres' database to create target DB)
db-create: ## Create database if not exists
	@if [ -z "${DATABASE_URL}" ]; then echo "DATABASE_URL not set"; exit 1; fi;
	@DB_NO_DB=$$(echo "${DATABASE_URL}" | sed 's!/[^/]*$$!/postgres!'); \
	TARGET_DB=$$(echo "${DATABASE_URL}" | sed -E 's!.*/([^/?]+).*!\1!'); \
	EXISTS=$$(psql "$$DB_NO_DB" -tAc "SELECT 1 FROM pg_database WHERE datname='$$TARGET_DB'"); \
	if [ "$$EXISTS" != "1" ]; then \
		echo "Creating database $$TARGET_DB"; \
		psql "$$DB_NO_DB" -c "CREATE DATABASE \"$$TARGET_DB\""; \
	else echo "Database $$TARGET_DB already exists"; fi;

# Drop and recreate public schema (danger!)
db-reset: ## Drop and recreate public schema
	@if [ -z "${DATABASE_URL}" ]; then echo "DATABASE_URL not set"; exit 1; fi;
	psql "${DATABASE_URL}" -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS=":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
