package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/agent/domain"
	"github.com/hildanku/xemarify/pkg/query"
)

var ErrEnrollmentTokenInvalid = errors.New("enrollment token is invalid or already used")

// ListFilter holds all filter and pagination options for listing agents.
// It embeds query.BaseFilter for the shared sort/pagination contract, and can be
// extended with agent-specific fields (e.g. status filter) without touching the
// shared package.
type ListFilter struct {
	query.BaseFilter
	// Add agent-specific filter fields here when needed.
	// e.g. Status domain.AgentStatus
}

// AgentRepository defines the persistence contract for the agent module.
type AgentRepository interface {
	// CreateEnrollmentToken persists a new one-time enrollment token.
	CreateEnrollmentToken(ctx context.Context, token string) error

	// CreateWithEnrollmentToken creates an agent and marks the enrollment token as used atomically.
	CreateWithEnrollmentToken(ctx context.Context, enrollmentToken string, agent *domain.Agent) error

	// GetBySecret looks up an agent by its runtime secret. Returns nil if not found.
	GetBySecret(ctx context.Context, secret string) (*domain.Agent, error)

	// UpdateLastSeen updates last_seen_at and sets status to ONLINE.
	UpdateLastSeen(ctx context.Context, agentID uuid.UUID) error

	// Create inserts a new agent into the database and returns its generated ID.
	Create(ctx context.Context, agent *domain.Agent) error

	// Update updates an existing agent's mutable fields (name, hostname, version, status).
	Update(ctx context.Context, agentId uuid.UUID, agent *domain.Agent) error

	// GetByID looks up an agent by its ID. Returns nil if not found.
	GetByID(ctx context.Context, agentId uuid.UUID) (*domain.Agent, error)

	// List returns a filtered, sorted, paginated slice of agents together with
	// the total number of rows that match the filter (ignoring limit/offset).
	List(ctx context.Context, filter ListFilter) ([]*domain.Agent, int, error)

	// Delete removes an agent from the database by its ID.
	Delete(ctx context.Context, agentId uuid.UUID) error
}
