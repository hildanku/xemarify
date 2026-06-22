package transport

import (
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/alert/domain"
)

type AlertResponse struct {
	ID             uuid.UUID `json:"id"`
	RuleID         uuid.UUID `json:"rule_id"`
	RuleName       string    `json:"rule_name"`
	Severity       string    `json:"severity"`
	CorrelationKey string    `json:"correlation_key"`
	TriggeredAt    time.Time `json:"triggered_at"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type AlertEventResponse struct {
	ID         uuid.UUID              `json:"id"`
	EventTime  time.Time              `json:"event_time"`
	ReceivedAt time.Time              `json:"received_at"`
	AgentID    uuid.UUID              `json:"agent_id"`
	Hostname   string                 `json:"hostname,omitempty"`
	SourceIP   string                 `json:"source_ip,omitempty"`
	InputType  string                 `json:"input_type,omitempty"`
	Facility   string                 `json:"facility,omitempty"`
	Severity   string                 `json:"severity,omitempty"`
	Category   string                 `json:"category,omitempty"`
	Message    string                 `json:"message"`
	Normalized map[string]interface{} `json:"normalized,omitempty"`
	Raw        string                 `json:"raw,omitempty"`
}

type AlertDetailResponse struct {
	Alert       *AlertResponse        `json:"alert"`
	Events      []*AlertEventResponse `json:"events"`
	Explanation *AlertExplanation     `json:"explanation,omitempty"`
}

type AlertExplanation struct {
	Matched        bool                   `json:"matched"`
	Reason         string                 `json:"reason"`
	CorrelationKey string                 `json:"correlation_key,omitempty"`
	EvaluatedAt    time.Time              `json:"evaluated_at"`
	Details        map[string]interface{} `json:"details,omitempty"`
}

// ListAlertsMetadata carries pagination info for a list response.
// COUNT(*) and total_pages have been removed in favour of keyset pagination.
type ListAlertsMetadata struct {
	// NextCursor is the opaque token to pass as ?cursor= on the next request.
	// An empty string means this is the last page.
	NextCursor string `json:"next_cursor"`

	// HasMore is a convenience boolean derived from NextCursor.
	HasMore bool `json:"has_more"`

	// Limit is the page size that was applied.
	Limit int `json:"limit"`
}

type ListAlertsResponse struct {
	Items    []*AlertResponse   `json:"items"`
	Metadata ListAlertsMetadata `json:"metadata"`
}

func ToAlertResponse(a *domain.Alert) *AlertResponse {
	return &AlertResponse{
		ID:             a.ID,
		RuleID:         a.RuleID,
		RuleName:       a.RuleName,
		Severity:       a.Severity,
		CorrelationKey: a.CorrelationKey,
		TriggeredAt:    a.TriggeredAt,
		Status:         a.Status,
		CreatedAt:      a.CreatedAt,
	}
}

func ToAlertEventResponse(e *domain.AlertEvent) *AlertEventResponse {
	return &AlertEventResponse{
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

func ToAlertExplanationResponse(explanation *domain.AlertExplanation) *AlertExplanation {
	if explanation == nil {
		return nil
	}

	return &AlertExplanation{
		Matched:        explanation.Matched,
		Reason:         explanation.Reason,
		CorrelationKey: explanation.CorrelationKey,
		EvaluatedAt:    explanation.EvaluatedAt,
		Details:        explanation.Details,
	}
}