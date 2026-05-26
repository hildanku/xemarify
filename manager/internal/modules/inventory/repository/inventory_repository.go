package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/inventory/domain"
)

// InventoryRepository defines the persistence contract for the inventory module.
type InventoryRepository interface {
	// Upsert inserts or updates the inventory snapshot for the given agent.
	Upsert(ctx context.Context, inv *domain.Inventory) error

	// GetByAgentID returns the latest inventory snapshot for the given agent.
	// Returns nil if no snapshot exists yet.
	GetByAgentID(ctx context.Context, agentID uuid.UUID) (*domain.Inventory, error)
}
