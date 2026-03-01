package repository

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/agent/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgAgentRepository struct {
	db *pgxpool.Pool
}

// NewPgAgentRepository creates a Postgres-backed AgentRepository.
func NewPgAgentRepository(db *pgxpool.Pool) AgentRepository {
	return &pgAgentRepository{db: db}
}

func (r *pgAgentRepository) GetByKey(ctx context.Context, key string) (*domain.Agent, error) {
	const q = `
		SELECT id, name, hostname, key, ip_address::text, version, status, created_at, last_seen_at
		FROM agents
		WHERE key = $1
		LIMIT 1
	`

	row := r.db.QueryRow(ctx, q, key)

	var a domain.Agent
	var ipAddress *string
	var lastSeenAt *time.Time

	err := row.Scan(
		&a.ID,
		&a.Name,
		&a.Hostname,
		&a.Key,
		&ipAddress,
		&a.Version,
		&a.Status,
		&a.CreatedAt,
		&lastSeenAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if ipAddress != nil {
		a.IPAddress = *ipAddress
	}
	a.LastSeenAt = lastSeenAt

	// Timing-safe key comparison to prevent timing attacks.
	if subtle.ConstantTimeCompare([]byte(a.Key), []byte(key)) != 1 {
		return nil, nil
	}

	return &a, nil
}

func (r *pgAgentRepository) UpdateLastSeen(ctx context.Context, agentID uuid.UUID) error {
	const q = `
		UPDATE agents
		SET last_seen_at = NOW(),
		    status = 'ONLINE'
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, q, agentID)
	return err
}
