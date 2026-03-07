package transport

import "time"

// IngestEventRequest is the JSON payload received from an agent via POST /api/v1/events.
// This is separated from the domain model to allow independent evolution of the HTTP contract.
type IngestEventRequest struct {
	ID         string                 `json:"id"         binding:"required,uuid"`
	EventTime  string                 `json:"event_time"`
	Message    string                 `json:"message"    binding:"required"`
	Raw        string                 `json:"raw"        binding:"required"`
	InputType  string                 `json:"input_type"`
	Facility   string                 `json:"facility"`
	Severity   string                 `json:"severity"`
	Category   string                 `json:"category"`
	Hostname   string                 `json:"hostname"`
	SourceIP   string                 `json:"source_ip"`
	Normalized map[string]interface{} `json:"normalized"`
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
	// Defaults to NOW()-24h server-side. Drives partition pruning.
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
