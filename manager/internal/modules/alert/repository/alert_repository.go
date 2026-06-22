package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/alert/domain"
	"github.com/hildanku/xemarify/pkg/query"
)

// ListFilter holds all filter and pagination options for listing alerts.
// Keyset pagination uses (triggered_at, id) as the composite cursor.
// COUNT(*) and OFFSET have been intentionally removed.
type ListFilter struct {
	query.BaseFilter
	Severity      string
	Status        string
	RuleID        *uuid.UUID
	TriggeredFrom *time.Time
	TriggeredTo   *time.Time

	// Cursor is the opaque keyset pagination token returned by the previous
	// List call. Empty string means "start from the first page".
	Cursor string
}

type AlertRepository interface {
	// List returns a filtered, sorted, paginated slice of alerts and an opaque
	// next-page cursor. The cursor is empty when there are no further pages.
	// COUNT(*) is intentionally omitted.
	List(ctx context.Context, filter ListFilter) ([]*domain.Alert, string, error)

	GetByID(ctx context.Context, id uuid.UUID) (*domain.AlertDetail, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (bool, error)
}