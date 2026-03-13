package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	AlertStatusNew          = "new"
	AlertStatusAcknowledged = "acknowledged"
	AlertStatusClosed       = "closed"
)

type Alert struct {
	ID             uuid.UUID
	RuleID         uuid.UUID
	RuleName       string
	Severity       string
	CorrelationKey string
	TriggeredAt    time.Time
	Status         string
	CreatedAt      time.Time
}

type AlertEvent struct {
	ID         uuid.UUID
	EventTime  time.Time
	ReceivedAt time.Time
	AgentID    uuid.UUID
	Hostname   string
	SourceIP   string
	InputType  string
	Facility   string
	Severity   string
	Category   string
	Message    string
	Normalized map[string]interface{}
	Raw        string
}

type AlertDetail struct {
	Alert  *Alert
	Events []*AlertEvent
}
