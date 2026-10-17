# AppTracker: one Go binary (API + embedded React build). Run `make help`.

GOBIN       := $(shell go env GOPATH)/bin
SQLC        := $(GOBIN)/sqlc
SWAG        := $(GOBIN)/swag
GOOSE       := $(GOBIN)/goose
AIR         := $(GOBIN)/air
MIGRATIONS  := internal/store/sqlite/migrations
SWAG_FLAGS  := -g cmd/apptracker/main.go --parseInternal --packageName docs
BINARY      := bin/apptracker

.DEFAULT_GOAL := help
.PHONY: help tools dev run build web generate sqlc swagger migrate-up migrate-down migrate-status migrate-new test check

help: ## List targets
	@grep -hE '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-16s %s\n", $$1, $$2}'

tools: ## Install sqlc, goose, swag and air (pinned), then npm ci
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
	go install github.com/pressly/goose/v3/cmd/goose@v3.29.0
	go install github.com/swaggo/swag/cmd/swag@v1.16.6
	go install github.com/air-verse/air@v1.64.5
	npm ci

dev: ## Go server with live reload (air, :8080) and the Vite dev server (:5173)
	@trap 'kill 0' EXIT INT TERM; $(AIR) & npm run dev & wait

run: ## Run the Go server without Vite (serves the last frontend build)
	go run ./cmd/apptracker

web: ## Vite build into internal/web/dist
	npm run build

build: web ## Vite build, then go build into bin/apptracker
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY) ./cmd/apptracker

generate: sqlc swagger ## Run sqlc and swag

sqlc: ## Generate typed queries from internal/store/sqlite/queries
	$(SQLC) generate

swagger: ## Generate the API spec into docs/ and format the annotations
	$(SWAG) fmt
	$(SWAG) init $(SWAG_FLAGS)

migrate-up: ## Apply pending migrations to DB_PATH (default data/app.db)
	go run ./cmd/apptracker migrate up

migrate-down: ## Roll back the latest migration
	go run ./cmd/apptracker migrate down

migrate-status: ## Show applied and pending migrations
	go run ./cmd/apptracker migrate status

migrate-new: ## Create a migration: make migrate-new name=add_things
	@test -n "$(name)" || (echo "usage: make migrate-new name=add_things" >&2; exit 1)
	$(GOOSE) -dir $(MIGRATIONS) -s create $(name) sql

test: ## Go tests
	go test ./...

check: ## go vet, gofmt, generated code up to date, no innerHTML in src/
	go vet ./...
	@test -z "$$(gofmt -l cmd internal)" || (echo "gofmt needed:"; gofmt -l cmd internal; exit 1)
	$(SQLC) diff
	@tmp=$$(mktemp -d); \
	$(SWAG) init $(SWAG_FLAGS) -o $$tmp --quiet && \
	diff -r docs $$tmp >/dev/null || { echo "docs/ is out of date: run make swagger" >&2; rm -rf $$tmp; exit 1; }; \
	rm -rf $$tmp
	@if grep -rnE 'dangerouslySetInnerHTML|innerHTML' src; then \
		echo "innerHTML / dangerouslySetInnerHTML is banned in src/ (render through React)" >&2; exit 1; fi
	@echo "check: ok"
