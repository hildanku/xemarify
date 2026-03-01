package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/agent/domain"
)

// AgentRepository defines the persistence contract for the agent module.
type AgentRepository interface {
	// GetByKey looks up an agent by its secret key. Returns nil if not found.
	GetByKey(ctx context.Context, key string) (*domain.Agent, error)

	// UpdateLastSeen updates last_seen_at and sets status to ONLINE.
	UpdateLastSeen(ctx context.Context, agentID uuid.UUID) error
}
