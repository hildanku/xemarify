package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/event/domain"
	"github.com/hildanku/xemarify/pkg/query"
)

// ListFilter holds filter and pagination options for listing events.
// It embeds query.BaseFilter for the shared sort/pagination contract.
//
// NOTE: Search is applied only on indexed columns (hostname, severity, category)
// to avoid sequential scans on the partitioned events table.
// See notes.txt for performance considerations and future improvements.
type ListFilter struct {
	query.BaseFilter

	// DateFrom restricts results to events received on or after this time.
	// Defaults to NOW()-24h if not set. Required for partition pruning.
	DateFrom *time.Time

	// DateTo restricts results to events received on or before this time.
	// Defaults to NOW() if not set.
	DateTo *time.Time

	// AgentID filters events from a specific agent (optional).
	AgentID *string

	// Severity filters by exact severity value (optional).
	Severity string

	// Category filters by exact category value (optional).
	Category string
}

// EventRepository defines the persistence contract for the event module.
type EventRepository interface {
	// Insert persists a single event into the partitioned events table.
	Insert(ctx context.Context, event *domain.Event) error

	// List returns a filtered, sorted, paginated slice of events together
	// with the total count matching the filter (ignoring limit/offset).
	List(ctx context.Context, filter ListFilter) ([]*domain.Event, int, error)

	// GetByID returns a single event by ID, or nil when not found.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Event, error)
}
