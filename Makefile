SHELL := /bin/bash
APP_NAME := shortlink
GO := go

.PHONY: dev run build test migrate seed help

dev: ## Run api in dev mode
	$(GO) run ./cmd/shortlink-api --config ./config/config.example.yaml

run: ## Run redirector
	$(GO) run ./cmd/redirector --config ./config/config.example.yaml

build: ## Build binaries
	$(GO) build -o bin/shortlink-api ./cmd/shortlink-api
	$(GO) build -o bin/redirector ./cmd/redirector

 test: ## Run tests
	$(GO) test ./... -count=1 -race -timeout=5m

migrate: ## Run migrations (requires golang-migrate installed)
	migrate -path migrations -database "$${DATABASE_URL}" up

seed: ## Seed admin
	$(GO) run ./cmd/shortlink-api seed --admin-email admin@example.com --password Admin@123 --tenant default --config ./config/config.example.yaml

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS=":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
