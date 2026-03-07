package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/audit/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgAuditLogRepository struct {
	db *pgxpool.Pool
}

// NewPgAuditLogRepository creates a Postgres-backed AuditLogRepository.
func NewPgAuditLogRepository(db *pgxpool.Pool) AuditLogRepository {
	return &pgAuditLogRepository{db: db}
}

func (r *pgAuditLogRepository) Create(ctx context.Context, e *domain.AuditLog) error {
	metaJSON, err := json.Marshal(e.Metadata)
	if err != nil {
		return err
	}

	const q = `
		INSERT INTO audit_logs
			(id, user_id, user_identifier, action, object_type, object_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = r.db.Exec(ctx, q,
		e.ID,
		e.UserID,
		e.UserIdentifier,
		e.Action,
		e.ObjectType,
		e.ObjectID,
		metaJSON,
		e.CreatedAt,
	)
	return err
}

func (r *pgAuditLogRepository) List(ctx context.Context, f ListFilter) ([]*domain.AuditLog, int, error) {
	allowedCols := map[string]string{
		"action":          "action",
		"user_identifier": "user_identifier",
		"created_at":      "created_at",
	}
	sortCol, ok := allowedCols[f.SortBy]
	if !ok {
		sortCol = "created_at"
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

	// Nullify empty action/search so the filter is skipped in SQL.
	var actionFilter *string
	if f.Action != "" {
		actionFilter = &f.Action
	}
	var searchFilter *string
	if f.Search != "" {
		searchFilter = &f.Search
	}

	baseWhere := `
		WHERE ($1::VARCHAR IS NULL OR action = $1)
		  AND ($2::TIMESTAMP IS NULL OR created_at >= $2)
		  AND ($3::TIMESTAMP IS NULL OR created_at <= $3)
		  AND ($4::VARCHAR IS NULL OR action ILIKE '%' || $4 || '%'
		       OR user_identifier ILIKE '%' || $4 || '%')
	`

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", baseWhere)
	var total int
	if err := r.db.QueryRow(ctx, countQ, actionFilter, f.DateFrom, f.DateTo, searchFilter).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQ := fmt.Sprintf(`
		SELECT id, user_id, user_identifier, action, object_type, object_id, metadata, created_at
		FROM audit_logs
		%s
		ORDER BY %s %s
		LIMIT $5 OFFSET $6
	`, baseWhere, sortCol, direction)

	rows, err := r.db.Query(ctx, listQ, actionFilter, f.DateFrom, f.DateTo, searchFilter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		entry, err := scanAuditLog(rows)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// ─── scanning helper ─────────────────────────────────────────────────────────

type auditScannable interface {
	Scan(dest ...interface{}) error
}

func scanAuditLog(s auditScannable) (*domain.AuditLog, error) {
	var e domain.AuditLog
	var metaBytes []byte
	var userID *uuid.UUID
	var objectID *uuid.UUID

	// pgx returns pgx.ErrNoRows for single rows; for multi-row scan we just
	// propagate the error.
	err := s.Scan(
		&e.ID,
		&userID,
		&e.UserIdentifier,
		&e.Action,
		&e.ObjectType,
		&objectID,
		&metaBytes,
		&e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	e.UserID = userID
	e.ObjectID = objectID

	if metaBytes != nil {
		if err := json.Unmarshal(metaBytes, &e.Metadata); err != nil {
			return nil, err
		}
	}
	return &e, nil
}
