package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/event/domain"
	"github.com/jackc/pgx/v5"
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

func (r *pgEventRepository) BatchInsert(ctx context.Context, events []*domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	rows := make([][]any, len(events))
	for i, e := range events {
		normalizedJSON, err := json.Marshal(e.Normalized)
		if err != nil {
			return fmt.Errorf("marshal normalized for event %s: %w", e.ID, err)
		}

		var sourceIP *string
		if e.SourceIP != "" {
			sourceIP = &e.SourceIP
		}

		rows[i] = []any{
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
			time.Now().UTC(),
		}
	}

	_, err := r.db.CopyFrom(
		ctx,
		pgx.Identifier{"events"},
		[]string{
			"id", "event_time", "received_at",
			"agent_id", "hostname", "source_ip", "input_type",
			"facility", "severity", "category",
			"message", "normalized", "raw",
			"created_at",
		},
		pgx.CopyFromRows(rows),
	)
	return err
}

// List returns a page of events using keyset (cursor) pagination.
//
// COUNT(*) and OFFSET have been removed intentionally: both cause full or
// near-full partition scans on large partitioned tables, which was the root
// cause of p95 > 4 s under load. The replacement uses a composite tuple
// comparison (received_at, id) that hits the covering index directly.
//
// Ordering is always by (received_at, id) in the requested direction so that
// the cursor position is unambiguous. Custom sort columns are not supported
// together with cursor pagination, received_at is the canonical sort key for
// time-series event data.
func (r *pgEventRepository) List(ctx context.Context, f ListFilter) ([]*domain.Event, string, error) {
	direction := "DESC"
	if strings.EqualFold(string(f.Order), "asc") {
		direction = "ASC"
	}

	limit := 10
	if f.Limit > 0 {
		limit = f.Limit
	}

	// Default date window: last 30 days. Keeps partition pruning effective.
	now := time.Now().UTC()
	dateFrom := now.Add(-30 * 24 * time.Hour)
	if f.DateFrom != nil {
		dateFrom = *f.DateFrom
	}
	dateTo := now
	if f.DateTo != nil {
		dateTo = *f.DateTo
	}

	// $1, $2, always present; drive partition pruning on received_at.
	args := []any{dateFrom, dateTo}
	conditions := []string{"received_at >= $1", "received_at <= $2"}

	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		n := len(args)
		conditions = append(conditions,
			fmt.Sprintf("(hostname ILIKE $%d OR severity ILIKE $%d OR category ILIKE $%d)", n, n, n),
		)
	}
	if f.Severity != "" {
		args = append(args, f.Severity)
		conditions = append(conditions, fmt.Sprintf("severity = $%d", len(args)))
	}
	if f.Category != "" {
		args = append(args, f.Category)
		conditions = append(conditions, fmt.Sprintf("category = $%d", len(args)))
	}
	if f.AgentID != nil && *f.AgentID != "" {
		args = append(args, *f.AgentID)
		conditions = append(conditions, fmt.Sprintf("agent_id = $%d", len(args)))
	}

	// Keyset condition: resume from the cursor position.
	// ROW comparison (received_at, id) < (ts, uuid) lets Postgres seek directly
	// into the index without scanning all preceding rows.
	if f.Cursor != "" {
		cur, err := DecodeCursor(f.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("list events: %w", err)
		}
		args = append(args, cur.ReceivedAt, cur.ID)
		nTs, nID := len(args)-1, len(args)
		op := "<"
		if direction == "ASC" {
			op = ">"
		}
		// Standard SQL row value comparison; supported natively by PostgreSQL.
		conditions = append(conditions,
			fmt.Sprintf("(received_at, id) %s ($%d, $%d)", op, nTs, nID),
		)
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	args = append(args, limit)
	nLimit := len(args)

	dataQ := fmt.Sprintf(`
		SELECT id, event_time, received_at,
		       agent_id, hostname, source_ip::text, input_type,
		       facility, severity, category,
		       message, normalized, raw
		FROM events
		%s
		ORDER BY received_at %s, id %s
		LIMIT $%d
	`, where, direction, direction, nLimit)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		var e domain.Event
		var sourceIP *string
		var metaBytes []byte

		if err := rows.Scan(
			&e.ID,
			&e.EventTime,
			&e.ReceivedAt,
			&e.AgentID,
			&e.Hostname,
			&sourceIP,
			&e.InputType,
			&e.Facility,
			&e.Severity,
			&e.Category,
			&e.Message,
			&metaBytes,
			&e.Raw,
		); err != nil {
			return nil, "", err
		}

		if sourceIP != nil {
			e.SourceIP = *sourceIP
		}
		if metaBytes != nil {
			if err := json.Unmarshal(metaBytes, &e.Normalized); err != nil {
				return nil, "", err
			}
		}
		events = append(events, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	// Build the next-page cursor from the last row.
	// An empty cursor signals that the caller has reached the end of results.
	nextCursor := ""
	if len(events) == limit {
		last := events[len(events)-1]
		nextCursor = EncodeCursor(PageCursor{
			ReceivedAt: last.ReceivedAt,
			ID:         last.ID,
		})
	}

	return events, nextCursor, nil
}

func (r *pgEventRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	const q = `
		SELECT id, event_time, received_at,
		       agent_id, hostname, source_ip::text, input_type,
		       facility, severity, category,
		       message, normalized, raw
		FROM events
		WHERE id = $1
		LIMIT 1
	`

	var event domain.Event
	var sourceIP *string
	var normalizedBytes []byte

	err := r.db.QueryRow(ctx, q, id).Scan(
		&event.ID,
		&event.EventTime,
		&event.ReceivedAt,
		&event.AgentID,
		&event.Hostname,
		&sourceIP,
		&event.InputType,
		&event.Facility,
		&event.Severity,
		&event.Category,
		&event.Message,
		&normalizedBytes,
		&event.Raw,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if sourceIP != nil {
		event.SourceIP = *sourceIP
	}
	if normalizedBytes != nil {
		if err := json.Unmarshal(normalizedBytes, &event.Normalized); err != nil {
			return nil, err
		}
	}

	return &event, nil
}
