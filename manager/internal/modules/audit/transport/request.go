package transport

import "time"

// ListAuditLogsQuery holds the query parameters for GET /api/v1/audit-logs.
type ListAuditLogsQuery struct {
	Action   string     `form:"action"`
	DateFrom *time.Time `form:"date_from" time_format:"2006-01-02T15:04:05Z07:00"`
	DateTo   *time.Time `form:"date_to"   time_format:"2006-01-02T15:04:05Z07:00"`
	Page     int        `form:"page,default=1"     binding:"omitempty,min=1"`
	PageSize int        `form:"page_size,default=20" binding:"omitempty,min=1,max=100"`
}
