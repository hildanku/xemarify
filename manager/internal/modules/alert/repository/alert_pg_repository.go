package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/alert/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgAlertRepository struct {
	db *pgxpool.Pool
}

func NewPgAlertRepository(db *pgxpool.Pool) AlertRepository {
	return &pgAlertRepository{db: db}
}

func (r *pgAlertRepository) List(ctx context.Context, f ListFilter) ([]*domain.Alert, int, error) {
	allowedCols := map[string]string{
		"triggered_at": "a.triggered_at",
		"severity":     "a.severity",
		"status":       "a.status",
		"created_at":   "a.created_at",
	}
	sortCol, ok := allowedCols[f.SortBy]
	if !ok {
		sortCol = "a.triggered_at"
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

	args := []any{}
	conditions := []string{}

	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		n := len(args)
		conditions = append(conditions, fmt.Sprintf("(r.name ILIKE $%d OR a.correlation_key ILIKE $%d)", n, n))
	}
	if f.Severity != "" {
		args = append(args, f.Severity)
		conditions = append(conditions, fmt.Sprintf("a.severity = $%d", len(args)))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", len(args)))
	}
	if f.RuleID != nil {
		args = append(args, *f.RuleID)
		conditions = append(conditions, fmt.Sprintf("a.rule_id = $%d", len(args)))
	}
	if f.TriggeredFrom != nil {
		args = append(args, *f.TriggeredFrom)
		conditions = append(conditions, fmt.Sprintf("a.triggered_at >= $%d", len(args)))
	}
	if f.TriggeredTo != nil {
		args = append(args, *f.TriggeredTo)
		conditions = append(conditions, fmt.Sprintf("a.triggered_at <= $%d", len(args)))
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQ := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM alerts a
		LEFT JOIN rules r ON r.id = a.rule_id
		%s
	`, where)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	nLimit, nOffset := len(args)-1, len(args)
	dataQ := fmt.Sprintf(`
		SELECT a.id, a.rule_id, COALESCE(r.name, ''), a.severity, a.correlation_key, a.triggered_at, a.status, a.created_at
		FROM alerts a
		LEFT JOIN rules r ON r.id = a.rule_id
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, sortCol, direction, nLimit, nOffset)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	alerts := make([]*domain.Alert, 0, limit)
	for rows.Next() {
		alert, err := scanAlert(rows)
		if err != nil {
			return nil, 0, err
		}
		alerts = append(alerts, alert)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}

func (r *pgAlertRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AlertDetail, error) {
	const alertQ = `
		SELECT a.id, a.rule_id, COALESCE(r.name, ''), a.severity, a.correlation_key, a.triggered_at, a.status, a.created_at
		FROM alerts a
		LEFT JOIN rules r ON r.id = a.rule_id
		WHERE a.id = $1
	`

	alert, err := scanAlert(r.db.QueryRow(ctx, alertQ, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	const eventsQ = `
		SELECT e.id, e.event_time, e.received_at,
		       e.agent_id, e.hostname, e.source_ip::text, e.input_type,
		       e.facility, e.severity, e.category,
		       e.message, e.normalized, e.raw
		FROM alert_events ae
		JOIN events e
		  ON e.id = ae.event_id
		 AND e.received_at = ae.received_at
		WHERE ae.alert_id = $1
		ORDER BY e.received_at DESC
	`

	rows, err := r.db.Query(ctx, eventsQ, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*domain.AlertEvent, 0, 8)
	for rows.Next() {
		var ev domain.AlertEvent
		var sourceIP *string
		var normalizedBytes []byte

		if err := rows.Scan(
			&ev.ID,
			&ev.EventTime,
			&ev.ReceivedAt,
			&ev.AgentID,
			&ev.Hostname,
			&sourceIP,
			&ev.InputType,
			&ev.Facility,
			&ev.Severity,
			&ev.Category,
			&ev.Message,
			&normalizedBytes,
			&ev.Raw,
		); err != nil {
			return nil, err
		}

		if sourceIP != nil {
			ev.SourceIP = *sourceIP
		}
		if normalizedBytes != nil {
			if err := json.Unmarshal(normalizedBytes, &ev.Normalized); err != nil {
				return nil, err
			}
		}

		events = append(events, &ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &domain.AlertDetail{Alert: alert, Events: events}, nil
}

func (r *pgAlertRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (bool, error) {
	const q = `
		UPDATE alerts
		SET status = $2
		WHERE id = $1
	`
	result, err := r.db.Exec(ctx, q, id, status)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAlert(row rowScanner) (*domain.Alert, error) {
	var alert domain.Alert
	if err := row.Scan(
		&alert.ID,
		&alert.RuleID,
		&alert.RuleName,
		&alert.Severity,
		&alert.CorrelationKey,
		&alert.TriggeredAt,
		&alert.Status,
		&alert.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &alert, nil
}
