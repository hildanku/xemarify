package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/user/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgUserRepository struct {
	db *pgxpool.Pool
}

// NewPgUserRepository creates a Postgres-backed UserRepository.
func NewPgUserRepository(db *pgxpool.Pool) UserRepository {
	return &pgUserRepository{db: db}
}

func (r *pgUserRepository) Create(ctx context.Context, u *domain.User) error {
	const q = `
		INSERT INTO users (id, username, email, role, avatar, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, q, u.ID, u.Username, u.Email, string(u.Role), u.Avatar, u.CreatedAt)
	return err
}

func (r *pgUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
		SELECT id, username, email, role, avatar, created_at, updated_at
		FROM users
		WHERE id = $1
		LIMIT 1
	`
	return r.scanOne(r.db.QueryRow(ctx, q, id))
}

func (r *pgUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `
		SELECT id, username, email, role, avatar, created_at, updated_at
		FROM users
		WHERE email = $1
		LIMIT 1
	`
	return r.scanOne(r.db.QueryRow(ctx, q, email))
}

func (r *pgUserRepository) List(ctx context.Context) ([]*domain.User, error) {
	const q = `
		SELECT id, username, email, role, avatar, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *pgUserRepository) Update(ctx context.Context, u *domain.User) error {
	const q = `
		UPDATE users
		SET username   = $2,
		    email      = $3,
		    role       = $4,
		    avatar     = $5,
		    updated_at = $6
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, q, u.ID, u.Username, u.Email, string(u.Role), u.Avatar, u.UpdatedAt)
	return err
}

func (r *pgUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}

func (r *pgUserRepository) scanOne(row pgx.Row) (*domain.User, error) {
	u, err := r.scanRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func (r *pgUserRepository) scanRow(s scannable) (*domain.User, error) {
	var u domain.User
	var role string
	var updatedAt *time.Time

	err := s.Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&role,
		&u.Avatar,
		&u.CreatedAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.Role = domain.Role(role)
	u.UpdatedAt = updatedAt
	return &u, nil
}
