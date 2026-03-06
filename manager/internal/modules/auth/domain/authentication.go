package domain

import (
	"time"

	"github.com/google/uuid"
)

// Authentication holds the credential record for a user.
type Authentication struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	PasswordHash string
	RefreshToken *string
	LastLoginAt  *time.Time
	CreatedAt    time.Time
}
