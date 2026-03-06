package repository

import (
	"context"
	"time"

	"github.com/hildanku/xemarify/internal/modules/audit/domain"
)

// ListFilter holds optional filters and pagination for audit log listing.
type ListFilter struct {
	Action   string
	DateFrom *time.Time
	DateTo   *time.Time
	Page     int
	PageSize int
}

// AuditLogRepository defines the persistence contract for audit logs.
type AuditLogRepository interface {
	// Create inserts a new audit log entry.
	Create(ctx context.Context, entry *domain.AuditLog) error

	// List returns paginated audit logs matching the optional filters.
	// Returns the matching entries, the total count, and any error.
	List(ctx context.Context, filter ListFilter) ([]*domain.AuditLog, int, error)
}
