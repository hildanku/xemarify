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

func (r *pgEventRepository) List(ctx context.Context, f ListFilter) ([]*domain.Event, int, error) {
	allowedCols := map[string]string{
		"received_at": "received_at",
		"event_time":  "event_time",
		"hostname":    "hostname",
		"severity":    "severity",
		"category":    "category",
		"created_at":  "created_at",
	}
	sortCol, ok := allowedCols[f.SortBy]
	if !ok {
		sortCol = "received_at"
	}

	direction := "DESC"
	if strings.EqualFold(string(f.Order), "asc") {
		direction = "ASC"
	}

	limit := 10
	if f.Limit > 0 {
		limit = f.Limit
	}
	offset := 0
	if f.Offset > 0 {
		offset = f.Offset
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

	// Build dynamic WHERE clause.
	args := []any{dateFrom, dateTo} // $1, $2 always present (partition pruning)
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

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Total count - bounded by the date range so Postgres only scans pruned partitions.
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM events %s", where)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Paginated data query.
	args = append(args, limit, offset)
	nLimit, nOffset := len(args)-1, len(args)
	dataQ := fmt.Sprintf(`
		SELECT id, event_time, received_at,
		       agent_id, hostname, source_ip::text, input_type,
		       facility, severity, category,
		       message, normalized, raw
		FROM events
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, sortCol, direction, nLimit, nOffset)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, err
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
			return nil, 0, err
		}

		if sourceIP != nil {
			e.SourceIP = *sourceIP
		}
		if metaBytes != nil {
			if err := json.Unmarshal(metaBytes, &e.Normalized); err != nil {
				return nil, 0, err
			}
		}
		events = append(events, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return events, total, nil
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
