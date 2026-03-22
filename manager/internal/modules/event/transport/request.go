package transport

import "time"

// IngestEvent contains a single normalized event in a batch ingest request.
type IngestEvent struct {
	EventTime  time.Time              `json:"event_time"`
	Hostname   string                 `json:"hostname" binding:"required"`
	SourceIP   string                 `json:"source_ip"`
	InputType  string                 `json:"input_type"`
	Facility   string                 `json:"facility"`
	Severity   string                 `json:"severity" binding:"required,oneof=INFO LOW MEDIUM HIGH CRITICAL"`
	Category   string                 `json:"category" binding:"required"`
	Message    string                 `json:"message" binding:"required"`
	Normalized map[string]interface{} `json:"normalized"`
	Raw        string                 `json:"raw" binding:"required"`
}

// EventBatchRequest is the payload received via POST /api/v1/events.
type EventBatchRequest struct {
	AgentID string        `json:"agent_id" binding:"required,uuid"`
	Events  []IngestEvent `json:"events" binding:"required,min=1,dive"`
}

// ListEventsQuery holds the query parameters for GET /api/v1/events.
type ListEventsQuery struct {
	// Search performs a case-insensitive partial match on hostname, severity, and category.
	// Intentionally limited to indexed columns \u2014 see notes.txt for details.
	Search string `form:"search"`

	// SortBy is the column to sort by: received_at, event_time, hostname, severity, category, created_at.
	SortBy string `form:"sort_by,default=received_at"`

	// Order is the sort direction: asc or desc.
	Order string `form:"order,default=desc" binding:"omitempty,oneof=asc desc"`

	// Limit is the maximum number of rows to return (1-100).
	Limit int `form:"limit,default=10" binding:"omitempty,min=1,max=100"`

	// Offset is the number of rows to skip.
	Offset int `form:"offset,default=0" binding:"omitempty,min=0"`

	// DateFrom restricts results to events received on or after this time.
	// Defaults to NOW()-30d server-side. Drives partition pruning.
	DateFrom *time.Time `form:"date_from" time_format:"2006-01-02T15:04:05Z07:00"`

	// DateTo restricts results to events received on or before this time.
	// Defaults to NOW() server-side.
	DateTo *time.Time `form:"date_to" time_format:"2006-01-02T15:04:05Z07:00"`

	// AgentID filters events from a specific agent (optional, UUID string).
	AgentID string `form:"agent_id"`

	// Severity filters by exact severity value (optional).
	Severity string `form:"severity"`

	// Category filters by exact category value (optional).
	Category string `form:"category"`
}
