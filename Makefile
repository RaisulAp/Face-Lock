COMPOSE := docker compose -f deploy/docker-compose.yml
API_DIR := apps/faceclock-api
WEB_DIR := apps/faceclock-web
INF_DIR := services/faceclock-inference

.PHONY: up down reset logs migrate-up migrate-down migrate-new db-status psql lint test fmt

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

reset:
	$(COMPOSE) down -v
	$(COMPOSE) up -d --build

logs:
	$(COMPOSE) logs -f $(s)

# migrate-up/down/db-status run the migrate CLI FROM THE HOST (via `go run`,
# build tag `pgx5` — golang-migrate's CLI ships with zero database drivers
# compiled in unless a driver tag is passed) against Postgres's published
# host port, rather than exec-ing into the faceclock-api container — the
# runtime image intentionally does not bundle the migrate CLI binary
# (Fase 0 § 2.6 keeps the API image to just its own compiled binary).
# faceclock-api itself still runs migrations automatically on startup in
# development (cmd/api/main.go); these targets are for manual/CI use.
# Note the URL scheme is `pgx5://`, not `postgres://` — that is the scheme
# golang-migrate's database/pgx/v5 driver registers itself under.
MIGRATE_ENV = . $(CURDIR)/deploy/.env 2>/dev/null; \
	DB_URL="pgx5://$${POSTGRES_USER:-faceclock}:$$POSTGRES_PASSWORD@localhost:$${POSTGRES_PORT:-5433}/$${POSTGRES_DB:-faceclock}?sslmode=disable";

migrate-up:
	@$(MIGRATE_ENV) cd $(API_DIR) && go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate -path migrations -database "$$DB_URL" up

migrate-down:
	@$(MIGRATE_ENV) cd $(API_DIR) && go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate -path migrations -database "$$DB_URL" down -all

migrate-new:
	@if [ -z "$(name)" ]; then echo "usage: make migrate-new name=create_users"; exit 1; fi
	cd $(API_DIR) && go run github.com/golang-migrate/migrate/v4/cmd/migrate create -ext sql -dir migrations -seq $(name)

db-status:
	@$(MIGRATE_ENV) cd $(API_DIR) && go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate -path migrations -database "$$DB_URL" version

psql:
	$(COMPOSE) exec postgres psql -U $${POSTGRES_USER:-faceclock} -d $${POSTGRES_DB:-faceclock}

lint:
	cd $(API_DIR) && go vet ./... && (command -v golangci-lint >/dev/null && golangci-lint run ./... || echo "golangci-lint not installed locally — CI runs it")
	cd $(WEB_DIR) && npm run lint && npm run typecheck
	cd $(INF_DIR) && (command -v ruff >/dev/null && ruff check . || echo "ruff not installed locally — CI runs it")

test:
	cd $(API_DIR) && go test -race ./...
	cd $(WEB_DIR) && npm run typecheck
	cd $(INF_DIR) && (command -v pytest >/dev/null && pytest || echo "pytest not installed locally — CI runs it")

fmt:
	cd $(API_DIR) && gofmt -l -w .
	cd $(WEB_DIR) && npm run format:write
	cd $(INF_DIR) && (command -v ruff >/dev/null && ruff format . || echo "ruff not installed locally — CI runs it")
