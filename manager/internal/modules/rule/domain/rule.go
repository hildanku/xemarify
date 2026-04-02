package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RuleCondition is the detection logic stored as JSONB in the rules table.
type RuleCondition struct {
	Type                  string   `json:"type,omitempty"`
	EventType             string   `json:"event_type,omitempty"`
	GroupBy               []string `json:"group_by"`
	Threshold             int      `json:"threshold,omitempty"`
	WindowSec             int      `json:"window_sec,omitempty"`
	Severity              string   `json:"severity,omitempty"`
	SequenceSteps         []string `json:"sequence_steps,omitempty"`
	CorrelationEventTypes []string `json:"correlation_event_types,omitempty"`
	MinDistinctEventTypes int      `json:"min_distinct_event_types,omitempty"`
	BaselineWindowSec     int      `json:"baseline_window_sec,omitempty"`
	SpikeFactor           float64  `json:"spike_factor,omitempty"`
	AnomalyMinCount       int      `json:"anomaly_min_count,omitempty"`
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
