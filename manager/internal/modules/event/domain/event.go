package domain

import (
	"time"

	"github.com/google/uuid"
)

// Event is the internal domain representation of an ingested event.
type Event struct {
	ID         uuid.UUID
	EventTime  time.Time
	ReceivedAt time.Time

	AgentID   uuid.UUID
	Hostname  string
	SourceIP  string
	InputType string

	Facility string
	Severity string
	Category string

	Message    string
	Normalized map[string]interface{}
	Raw        string
}
