package repository

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
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

func (r *pgAgentRepository) CreateEnrollmentToken(ctx context.Context, token string) error {
	const q = `
		INSERT INTO agent_keys (id, key, status, created_at)
		VALUES ($1, $2, 'unused', NOW())
	`

	_, err := r.db.Exec(ctx, q, uuid.New(), token)
	return err
}

func (r *pgAgentRepository) CreateWithEnrollmentToken(ctx context.Context, enrollmentToken string, a *domain.Agent) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const lockKeyQuery = `
		SELECT id
		FROM agent_keys
		WHERE key = $1 AND status = 'unused'
		FOR UPDATE
	`

	var enrollmentID uuid.UUID
	if err := tx.QueryRow(ctx, lockKeyQuery, enrollmentToken).Scan(&enrollmentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEnrollmentTokenInvalid
		}
		return err
	}

	const insertAgentQuery = `
		INSERT INTO agents (
			id, name, hostname, key, ip_address, version, status, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		`

	var ipAddress *string
	if a.IPAddress != "" {
		ipAddress = &a.IPAddress
	}

	if _, err := tx.Exec(ctx, insertAgentQuery, a.ID, a.Name, a.Hostname, a.Secret, ipAddress, a.Version, a.Status); err != nil {
		return err
	}

	const useKeyQuery = `
		UPDATE agent_keys
		SET status = 'used',
		    used_by_agent_id = $2,
		    used_at = NOW()
		WHERE id = $1
	`

	if _, err := tx.Exec(ctx, useKeyQuery, enrollmentID, a.ID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (r *pgAgentRepository) GetBySecret(ctx context.Context, secret string) (*domain.Agent, error) {
	const q = `
		SELECT id, name, hostname, key, ip_address::text, version, status, created_at, last_seen_at
		FROM agents
		WHERE key = $1
		LIMIT 1
	`

	row := r.db.QueryRow(ctx, q, secret)

	var a domain.Agent
	var ipAddress *string
	var lastSeenAt *time.Time

	err := row.Scan(
		&a.ID,
		&a.Name,
		&a.Hostname,
		&a.Secret,
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

	// Timing-safe secret comparison to prevent timing attacks.
	if subtle.ConstantTimeCompare([]byte(a.Secret), []byte(secret)) != 1 {
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

func (r *pgAgentRepository) GetByID(ctx context.Context, agentId uuid.UUID) (*domain.Agent, error) {
	const q = `
		SELECT id, name, hostname, key, ip_address::text, version, status, created_at, last_seen_at
		FROM agents
		WHERE id = $1
		LIMIT 1
	`

	row := r.db.QueryRow(ctx, q, agentId)

	var a domain.Agent
	var ipAddress *string
	var lastSeenAt *time.Time

	err := row.Scan(
		&a.ID,
		&a.Name,
		&a.Hostname,
		&a.Secret,
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
	return &a, nil
}

func (r *pgAgentRepository) List(ctx context.Context, filter ListFilter) ([]*domain.Agent, int, error) {
	allowedCols := map[string]string{
		"name":         "name",
		"hostname":     "hostname",
		"status":       "status",
		"created_at":   "created_at",
		"last_seen_at": "last_seen_at",
		"version":      "version",
	}
	sortCol, ok := allowedCols[filter.SortBy]
	if !ok {
		sortCol = "created_at"
	}

	direction := "ASC"
	if strings.EqualFold(string(filter.Order), "desc") {
		direction = "DESC"
	}

	limit := 10
	if filter.Limit > 0 {
		limit = filter.Limit
	}
	offset := 0
	if filter.Offset > 0 {
		offset = filter.Offset
	}

	// Build optional WHERE clause.
	args := []any{}
	whereClause := ""
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		whereClause = "WHERE (name ILIKE $1 OR hostname ILIKE $1 OR ip_address::text ILIKE $1)"
	}

	// Total count (ignores limit/offset).
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM agents %s", whereClause)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Paginated data query.
	nextIdx := len(args) + 1
	args = append(args, limit, offset)
	dataQ := fmt.Sprintf(`
		SELECT id, name, hostname, key, ip_address::text, version, status, created_at, last_seen_at
		FROM agents
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortCol, direction, nextIdx, nextIdx+1)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var agents []*domain.Agent
	for rows.Next() {
		var a domain.Agent
		var ipAddress *string
		var lastSeenAt *time.Time

		if err := rows.Scan(
			&a.ID,
			&a.Name,
			&a.Hostname,
			&a.Secret,
			&ipAddress,
			&a.Version,
			&a.Status,
			&a.CreatedAt,
			&lastSeenAt,
		); err != nil {
			return nil, 0, err
		}

		if ipAddress != nil {
			a.IPAddress = *ipAddress
		}
		a.LastSeenAt = lastSeenAt
		agents = append(agents, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return agents, total, nil
}

func (r *pgAgentRepository) Create(ctx context.Context, a *domain.Agent) error {
	const q = `
		INSERT INTO agents (
			id, name, hostname, key, ip_address, version, status, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		`

	var ipAddress *string
	if a.IPAddress != "" {
		ipAddress = &a.IPAddress
	}
	_, err := r.db.Exec(ctx, q, a.ID, a.Name, a.Hostname, a.Secret, ipAddress, a.Version, a.Status)
	return err
}

func (r *pgAgentRepository) Update(ctx context.Context, agentId uuid.UUID, a *domain.Agent) error {
	const q = `
		UPDATE agents SET 
			name = $2,
			hostname = $3,
			ip_address = $4,
			version = $5,
			status = $6
		WHERE id = $1
	`

	var ipAddress *string
	if a.IPAddress != "" {
		ipAddress = &a.IPAddress
	}
	_, err := r.db.Exec(ctx, q, agentId, a.Name, a.Hostname, ipAddress, a.Version, a.Status)
	return err
}

func (r *pgAgentRepository) Delete(ctx context.Context, agentId uuid.UUID) error {
	const q = `DELETE FROM agents WHERE id = $1`
	_, err := r.db.Exec(ctx, q, agentId)
	return err
}
