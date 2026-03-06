package repository

import (
	"context"
	"encoding/json"
	"errors"

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
	// Default pagination
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	offset := (f.Page - 1) * f.PageSize

	// Nullify empty action string so the filter is skipped in SQL
	var actionFilter *string
	if f.Action != "" {
		actionFilter = &f.Action
	}

	const countQ = `
		SELECT COUNT(*) FROM audit_logs
		WHERE ($1::VARCHAR IS NULL OR action = $1)
		  AND ($2::TIMESTAMP IS NULL OR created_at >= $2)
		  AND ($3::TIMESTAMP IS NULL OR created_at <= $3)
	`
	var total int
	if err := r.db.QueryRow(ctx, countQ, actionFilter, f.DateFrom, f.DateTo).Scan(&total); err != nil {
		return nil, 0, err
	}

	const listQ = `
		SELECT id, user_id, user_identifier, action, object_type, object_id, metadata, created_at
		FROM audit_logs
		WHERE ($1::VARCHAR IS NULL OR action = $1)
		  AND ($2::TIMESTAMP IS NULL OR created_at >= $2)
		  AND ($3::TIMESTAMP IS NULL OR created_at <= $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.db.Query(ctx, listQ, actionFilter, f.DateFrom, f.DateTo, f.PageSize, offset)
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
