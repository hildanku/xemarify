package repository

import (
	"context"
	"time"

	"github.com/hildanku/xemarify/internal/modules/audit/domain"
	"github.com/hildanku/xemarify/pkg/query"
)

// ListFilter holds filter and pagination options for listing audit logs.
// It embeds query.BaseFilter for the shared sort/pagination contract and adds
// audit-specific filters (Action, DateFrom, DateTo).
type ListFilter struct {
	query.BaseFilter

	// Action filters by exact action string (optional).
	Action string

	// DateFrom filters entries created on or after this time (optional).
	DateFrom *time.Time

	// DateTo filters entries created on or before this time (optional).
	DateTo *time.Time
}

// AuditLogRepository defines the persistence contract for audit logs.
type AuditLogRepository interface {
	// Create inserts a new audit log entry.
	Create(ctx context.Context, entry *domain.AuditLog) error

	// List returns paginated audit logs matching the optional filters.
	// Returns the matching entries, the total count, and any error.
	List(ctx context.Context, filter ListFilter) ([]*domain.AuditLog, int, error)
}
