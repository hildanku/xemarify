package transport

import (
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/rule/domain"
)

// RuleConditionResponse is the JSON representation of a rule condition.
type RuleConditionResponse struct {
	EventType string   `json:"event_type"`
	GroupBy   []string `json:"group_by"`
	Threshold int      `json:"threshold"`
	WindowSec int      `json:"window_sec"`
	Severity  string   `json:"severity,omitempty"`
}

// RuleResponse is the JSON representation of a rule returned to the client.
type RuleResponse struct {
	ID          uuid.UUID             `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Level       string                `json:"level"`
	Enabled     bool                  `json:"enabled"`
	Condition   RuleConditionResponse `json:"condition"`
	Tags        []string              `json:"tags"`
	Version     int                   `json:"version"`
	CreatedBy   *uuid.UUID            `json:"created_by,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// ListRulesMetadata carries pagination and count information.
type ListRulesMetadata struct {
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
	Limit      int `json:"limit"`
	Offset     int `json:"offset"`
}

// ListRulesResponse wraps a paginated list of rules with metadata.
type ListRulesResponse struct {
	Items    []*RuleResponse   `json:"items"`
	Metadata ListRulesMetadata `json:"metadata"`
}

// ToRuleResponse converts a domain Rule to its HTTP response representation.
func ToRuleResponse(r *domain.Rule) *RuleResponse {
	tags := r.Tags
	if tags == nil {
		tags = []string{}
	}

	return &RuleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Level:       r.Level,
		Enabled:     r.Enabled,
		Condition: RuleConditionResponse{
			EventType: r.Condition.EventType,
			GroupBy:   r.Condition.GroupBy,
			Threshold: r.Condition.Threshold,
			WindowSec: r.Condition.WindowSec,
			Severity:  r.Condition.Severity,
		},
		Tags:      tags,
		Version:   r.Version,
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
