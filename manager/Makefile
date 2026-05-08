DB_URL=postgres://xx:xx@localhost:xx/xx?sslmode=disable

BINARY=bin/manager

# Build the manager binary
build:
	go build -o $(BINARY) ./cmd/...

# Run the manager (requires .env in manager/)
run: build
	./$(BINARY)

# Run directly without building binary
dev:
	go run ./cmd/...

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1

migrate-force:
	migrate -path migrations -database "$(DB_URL)" force $(version)

db-reset:
	migrate -path migrations -database "$(DB_URL)" drop -f
	migrate -path migrations -database "$(DB_URL)" up

# Seed default users (MANAGER / ANALYST / VIEWER) - idempotent, safe to re-run
seed:
	go run ./cmd/seed

.PHONY: build run dev migrate-create migrate-up migrate-down migrate-force db-reset seed