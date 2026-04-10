DB_URL=postgres://xemarify_manager:xemarify_manager@localhost:5445/xemarify_manager?sslmode=disable

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

# Create a new migration file (usage: make migrate-create name=create_users_table)
migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

# Run all up migrations
migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

# Rollback the last migration
migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1

# Force a specific version (usage: make migrate-force version=8)
migrate-force:
	migrate -path migrations -database "$(DB_URL)" force $(version)

# Clean/Reset DB (Hapus semua tabel dan migrasi ulang dari awal)
db-reset:
	migrate -path migrations -database "$(DB_URL)" drop -f
	migrate -path migrations -database "$(DB_URL)" up

# Seed default users (MANAGER / ANALYST / VIEWER) - idempotent, safe to re-run
seed:
	go run ./cmd/seed

.PHONY: build run dev migrate-create migrate-up migrate-down migrate-force db-reset seed
