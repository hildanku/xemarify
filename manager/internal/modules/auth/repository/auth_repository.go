package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/auth/domain"
)

// AuthRepository defines the persistence contract for authentication credentials.
type AuthRepository interface {
	// Create inserts a new authentication record.
	Create(ctx context.Context, a *domain.Authentication) error

	// GetByUserID fetches the authentication record for the given user.
	// Returns nil if not found.
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Authentication, error)

	// GetByRefreshToken fetches the authentication record matching the given
	// refresh token. Returns nil if not found.
	GetByRefreshToken(ctx context.Context, token string) (*domain.Authentication, error)

	// SetRefreshToken updates the stored refresh token.
	// Pass nil to clear the token (logout).
	SetRefreshToken(ctx context.Context, userID uuid.UUID, token *string) error

	// UpdateLastLogin stamps last_login_at with the current time.
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error
}
