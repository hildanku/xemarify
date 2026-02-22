package config

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDatabase(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5445,
		User:     "xemarify_manager",
		Password: "xemarify_manager",
		Name:     "xemarify_manager",
		SSLMode:  "disable",
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
	)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed connect: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}
