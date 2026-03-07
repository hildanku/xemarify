package transport

import "time"

// ListAuditLogsQuery holds the query parameters for GET /api/v1/audit-logs.
type ListAuditLogsQuery struct {
	// Action filters by exact action value (optional).
	Action string `form:"action"`

	// DateFrom filters entries created on or after this time (optional).
	DateFrom *time.Time `form:"date_from" time_format:"2006-01-02T15:04:05Z07:00"`

	// DateTo filters entries created on or before this time (optional).
	DateTo *time.Time `form:"date_to" time_format:"2006-01-02T15:04:05Z07:00"`

	// Search performs a case-insensitive partial match on action and user_identifier.
	Search string `form:"search"`

	// SortBy is the column to sort by: action, user_identifier, created_at.
	SortBy string `form:"sort_by,default=created_at"`

	// Order is the sort direction: asc or desc.
	Order string `form:"order,default=desc" binding:"omitempty,oneof=asc desc"`

	// Limit is the maximum number of rows to return (1-100).
	Limit int `form:"limit,default=10" binding:"omitempty,min=1,max=100"`

	// Offset is the number of rows to skip.
	Offset int `form:"offset,default=0" binding:"omitempty,min=0"`
}
