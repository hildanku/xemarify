package repository

import (
	"context"
	"encoding/json"

	"github.com/hildanku/xemarify/internal/modules/event/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgEventRepository struct {
	db *pgxpool.Pool
}

// NewPgEventRepository creates a Postgres-backed EventRepository.
func NewPgEventRepository(db *pgxpool.Pool) EventRepository {
	return &pgEventRepository{db: db}
}

func (r *pgEventRepository) Insert(ctx context.Context, e *domain.Event) error {
	normalizedJSON, err := json.Marshal(e.Normalized)
	if err != nil {
		return err
	}

	const q = `
		INSERT INTO events (
			id, event_time, received_at,
			agent_id, hostname, source_ip, input_type,
			facility, severity, category,
			message, normalized, raw,
			created_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6::inet, $7,
			$8, $9, $10,
			$11, $12, $13,
			NOW()
		)
	`

	var sourceIP *string
	if e.SourceIP != "" {
		sourceIP = &e.SourceIP
	}

	_, err = r.db.Exec(ctx, q,
		e.ID,
		e.EventTime,
		e.ReceivedAt,
		e.AgentID,
		e.Hostname,
		sourceIP,
		e.InputType,
		e.Facility,
		e.Severity,
		e.Category,
		e.Message,
		normalizedJSON,
		e.Raw,
	)
	return err
}
