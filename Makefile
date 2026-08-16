# ─── Variables ────────────────────────────────────────────────────────────────
APP_NAME  := task_tracker
CMD_PATH  := ./cmd/api
BIN_DIR   := ./bin
BIN       := $(BIN_DIR)/$(APP_NAME)
MIGRATIONS := ./migrations

# Local MySQL
DB_USER  ?= root
DB_PASS  ?= root
DB_HOST  ?= 127.0.0.1
DB_PORT  ?= 3306
DB_NAME  ?= task_tracker
LOCAL_DSN := $(DB_USER):$(DB_PASS)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)

# MySQL inside docker compose
DOCKER_DB_USER ?= task_user
DOCKER_DB_PASS ?= task_pass
DOCKER_DB_HOST ?= mysql
DOCKER_DB_PORT ?= 3306
DOCKER_DB_NAME ?= task_tracker
DOCKER_DSN     := $(DOCKER_DB_USER):$(DOCKER_DB_PASS)@tcp($(DOCKER_DB_HOST):$(DOCKER_DB_PORT))/$(DOCKER_DB_NAME)

# ─── Build ────────────────────────────────────────────────────────────────────
.PHONY: build
build:
	go build -o $(BIN) $(CMD_PATH)

.PHONY: run
run: build
	if [ -f .env ]; then export $$(cat .env | grep -v '^#' | xargs); fi; $(BIN)

.PHONY: clean
clean:
	rm -rf $(BIN_DIR) ./docs/swagger.*

# ─── Test ─────────────────────────────────────────────────────────────────────
.PHONY: test-access
test-access:
	go test ./internal/service -v

.PHONY: test-integration
test-integration:
	INTEGRATION_TEST=1 go test ./internal/service -v

# ─── Migrations (local MySQL) ────────────────────────────────────────────────
# Requires: go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
.PHONY: migrate-up
migrate-up:
	migrate -path $(MIGRATIONS) -database "mysql://$(LOCAL_DSN)?multiStatements=true" up

.PHONY: migrate-down
migrate-down:
	migrate -path $(MIGRATIONS) -database "mysql://$(LOCAL_DSN)?multiStatements=true" down

.PHONY: migrate-fix
migrate-fix:
	migrate -path $(MIGRATIONS) -database "mysql://$(LOCAL_DSN)?multiStatements=true" force 1

# ─── Migrations (MySQL inside docker compose) ────────────────────────────────
.PHONY: docker-migrate-up
docker-migrate-up:
	docker compose run --rm api migrate -path /api/migrations -database "mysql://$(DOCKER_DSN)?multiStatements=true" up

.PHONY: docker-migrate-down
docker-migrate-down:
	docker compose run --rm api migrate -path /api/migrations -database "mysql://$(DOCKER_DSN)?multiStatements=true" down

# ─── Local dev workflow ───────────────────────────────────────────────────────
# Spin up MySQL + Redis only (no api container). Run the Go app natively so
# Delve / IDE debuggers attach without ptrace gymnastics.
.PHONY: dev-infra
dev-infra:
	docker compose -f docker-compose.dev.yml up -d

.PHONY: dev-down
dev-down:
	docker compose -f docker-compose.dev.yml down

# Run under Delve for interactive debugging (breakpoints, step-through).
# Requires: go install github.com/go-delve/delve/cmd/dlv@latest
.PHONY: debug
debug:
	if [ -f .env ]; then export $$(cat .env | grep -v '^#' | xargs); fi; dlv debug $(CMD_PATH)

# ─── Docker Compose ───────────────────────────────────────────────────────────
.PHONY: docker-up
docker-up:
	docker compose up --build

.PHONY: docker-down
docker-down:
	docker compose down

.PHONY: docker-logs
docker-logs:
	docker compose logs -f

# ─── Swagger ──────────────────────────────────────────────────────────────────
# Requires: go install github.com/swaggo/swag/cmd/swag@latest
.PHONY: swagger
swagger:
	swag init -d ./cmd/api,./internal --parseDependency --parseInternal -o ./docs/swagger

# ─── Lint ─────────────────────────────────────────────────────────────────────
# Requires: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
.PHONY: lint
lint:
	golangci-lint run ./...

.DEFAULT_GOAL := build
