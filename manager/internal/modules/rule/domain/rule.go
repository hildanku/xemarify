package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RuleCondition is the detection logic stored as JSONB in the rules table.
type RuleCondition struct {
	EventType string   `json:"event_type"`
	GroupBy   []string `json:"group_by"`
	Threshold int      `json:"threshold"`
	WindowSec int      `json:"window_sec"`
	Severity  string   `json:"severity,omitempty"`
}

// Rule is the domain representation of a detection rule.
type Rule struct {
	ID          uuid.UUID
	Name        string
	Description string
	Level       string // mapped to severity enum
	Enabled     bool
	Condition   RuleCondition
	Tags        []string
	Version     int
	CreatedBy   *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ConditionJSON converts Condition to raw JSON for DB persistence.
func (r *Rule) ConditionJSON() (json.RawMessage, error) {
	return json.Marshal(r.Condition)
}
