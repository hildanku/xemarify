package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/user/domain"
	"github.com/hildanku/xemarify/pkg/query"
)

// ListFilter holds filter and pagination options for listing users.
// It embeds query.BaseFilter for the shared sort/pagination contract.
// Domain-specific filters (e.g. Role) can be added here without touching the shared package.
type ListFilter struct {
	query.BaseFilter
}

// UserRepository defines the persistence contract for the user module.
type UserRepository interface {
	// Create inserts a new user into the database.
	Create(ctx context.Context, user *domain.User) error

	// GetByID looks up a user by primary key. Returns nil if not found.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// GetByEmail looks up a user by email. Returns nil if not found.
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// List returns a filtered, sorted, paginated slice of users together with
	// the total number of rows that match the filter (ignoring limit/offset).
	List(ctx context.Context, filter ListFilter) ([]*domain.User, int, error)

	// Update updates mutable user fields (username, email, role, avatar, updated_at).
	Update(ctx context.Context, user *domain.User) error

	// Delete removes a user by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}
