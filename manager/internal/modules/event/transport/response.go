package transport

import (
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/event/domain"
)

// EventResponse is the JSON representation of a single event.
type EventResponse struct {
	ID         uuid.UUID              `json:"id"`
	EventTime  time.Time              `json:"event_time"`
	ReceivedAt time.Time              `json:"received_at"`
	AgentID    uuid.UUID              `json:"agent_id"`
	Hostname   string                 `json:"hostname"`
	SourceIP   string                 `json:"source_ip,omitempty"`
	InputType  string                 `json:"input_type,omitempty"`
	Facility   string                 `json:"facility,omitempty"`
	Severity   string                 `json:"severity,omitempty"`
	Category   string                 `json:"category,omitempty"`
	Message    string                 `json:"message"`
	Normalized map[string]interface{} `json:"normalized,omitempty"`
}

// ListEventsMetadata carries pagination and count info for a list response.
type ListEventsMetadata struct {
	// Total is the count of matching events within the requested date window.
	Total int `json:"total"`

	// TotalPages is derived from total and limit.
	TotalPages int `json:"total_pages"`

	// Limit is the page size that was applied.
	Limit int `json:"limit"`

	// Offset is the number of rows skipped.
	Offset int `json:"offset"`
}

// ListEventsResponse wraps a paginated slice of events and pagination metadata.
type ListEventsResponse struct {
	Items    []*EventResponse   `json:"items"`
	Metadata ListEventsMetadata `json:"metadata"`
}

// ToEventResponse converts a domain Event to its HTTP response form.
// Raw is intentionally omitted from the list response to keep payloads small.
func ToEventResponse(e *domain.Event) *EventResponse {
	return &EventResponse{
		ID:         e.ID,
		EventTime:  e.EventTime,
		ReceivedAt: e.ReceivedAt,
		AgentID:    e.AgentID,
		Hostname:   e.Hostname,
		SourceIP:   e.SourceIP,
		InputType:  e.InputType,
		Facility:   e.Facility,
		Severity:   e.Severity,
		Category:   e.Category,
		Message:    e.Message,
		Normalized: e.Normalized,
	}
}
