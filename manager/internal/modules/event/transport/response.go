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

// EventDetailResponse is the JSON representation used by GET /api/v1/events/:id.
// Includes raw payload for deep inspection.
type EventDetailResponse struct {
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
	Raw        string                 `json:"raw,omitempty"`
}

// ListEventsMetadata carries pagination info for a list response.
// COUNT(*) and total_pages have been removed to eliminate full-partition scans.
// Clients should paginate by passing the next_cursor value on subsequent requests.
type ListEventsMetadata struct {
	// NextCursor is the opaque token to pass as ?cursor= on the next request.
	// An empty string means this is the last page.
	NextCursor string `json:"next_cursor"`

	// HasMore is a convenience boolean derived from NextCursor.
	HasMore bool `json:"has_more"`

	// Limit is the page size that was applied.
	Limit int `json:"limit"`
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

// ToEventDetailResponse converts a domain Event to its detail response form.
func ToEventDetailResponse(e *domain.Event) *EventDetailResponse {
	return &EventDetailResponse{
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
		Raw:        e.Raw,
	}
}
