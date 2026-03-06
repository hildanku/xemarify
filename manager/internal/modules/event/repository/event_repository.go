package repository

import (
	"context"

	"github.com/hildanku/xemarify/internal/modules/event/domain"
)

// EventRepository defines the persistence contract for the event module.
type EventRepository interface {
	// Insert persists a single event into the partitioned events table.
	Insert(ctx context.Context, event *domain.Event) error
}
