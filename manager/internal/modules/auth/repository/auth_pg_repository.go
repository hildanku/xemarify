package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/auth/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgAuthRepository struct {
	db *pgxpool.Pool
}

// NewPgAuthRepository creates a Postgres-backed AuthRepository.
func NewPgAuthRepository(db *pgxpool.Pool) AuthRepository {
	return &pgAuthRepository{db: db}
}

func (r *pgAuthRepository) Create(ctx context.Context, a *domain.Authentication) error {
	const q = `
		INSERT INTO authentications (id, user_id, password_hash, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.Exec(ctx, q, a.ID, a.UserID, a.PasswordHash, a.CreatedAt)
	return err
}

func (r *pgAuthRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Authentication, error) {
	const q = `
		SELECT id, user_id, password_hash, refresh_token, last_login_at, created_at
		FROM authentications
		WHERE user_id = $1
		LIMIT 1
	`
	return r.scanOne(r.db.QueryRow(ctx, q, userID))
}

func (r *pgAuthRepository) GetByRefreshToken(ctx context.Context, token string) (*domain.Authentication, error) {
	const q = `
		SELECT id, user_id, password_hash, refresh_token, last_login_at, created_at
		FROM authentications
		WHERE refresh_token = $1
		LIMIT 1
	`
	return r.scanOne(r.db.QueryRow(ctx, q, token))
}

func (r *pgAuthRepository) SetRefreshToken(ctx context.Context, userID uuid.UUID, token *string) error {
	const q = `UPDATE authentications SET refresh_token = $2 WHERE user_id = $1`
	_, err := r.db.Exec(ctx, q, userID, token)
	return err
}

func (r *pgAuthRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	const q = `UPDATE authentications SET last_login_at = NOW() WHERE user_id = $1`
	_, err := r.db.Exec(ctx, q, userID)
	return err
}

// ─── scanning helpers ────────────────────────────────────────────────────────

func (r *pgAuthRepository) scanOne(row pgx.Row) (*domain.Authentication, error) {
	var a domain.Authentication
	var lastLoginAt *time.Time

	err := row.Scan(
		&a.ID,
		&a.UserID,
		&a.PasswordHash,
		&a.RefreshToken,
		&lastLoginAt,
		&a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	a.LastLoginAt = lastLoginAt
	return &a, nil
}
