package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CompiledRule is the runtime representation of a detection rule.
// Rules are precompiled at startup and indexed by event type for fast lookups.
type CompiledRule struct {
	ID                    uuid.UUID
	Type                  string
	EventType             string
	Threshold             int
	Window                time.Duration
	GroupBy               []string
	Severity              string
	SequenceSteps         []string
	CorrelationEventTypes []string
	MinDistinctEventTypes int
	BaselineWindow        time.Duration
	SpikeFactor           float64
	AnomalyMinCount       int
}

type storedRule struct {
	ID        uuid.UUID
	Level     string
	Condition []byte
}

type ruleCondition struct {
	Type                  string   `json:"type"`
	EventType             string   `json:"event_type"`
	GroupBy               []string `json:"group_by"`
	Threshold             int      `json:"threshold"`
	WindowSec             int      `json:"window_sec"`
	Severity              string   `json:"severity"`
	SequenceSteps         []string `json:"sequence_steps"`
	CorrelationEventTypes []string `json:"correlation_event_types"`
	MinDistinctEventTypes int      `json:"min_distinct_event_types"`
	BaselineWindowSec     int      `json:"baseline_window_sec"`
	SpikeFactor           float64  `json:"spike_factor"`
	AnomalyMinCount       int      `json:"anomaly_min_count"`
}

type RuleLoader interface {
	LoadEnabledRules(ctx context.Context) ([]storedRule, error)
}

type PGRuleLoader struct {
	db *pgxpool.Pool
}

func NewPGRuleLoader(db *pgxpool.Pool) *PGRuleLoader {
	return &PGRuleLoader{db: db}
}

func (l *PGRuleLoader) LoadEnabledRules(ctx context.Context) ([]storedRule, error) {
	const q = `
		SELECT id, level::text, condition
		FROM rules
		WHERE enabled = TRUE
	`

	rows, err := l.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]storedRule, 0, 64)
	for rows.Next() {
		var rule storedRule
		if err := rows.Scan(&rule.ID, &rule.Level, &rule.Condition); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rules, nil
}

type RuleCompiler struct{}

func NewRuleCompiler() *RuleCompiler {
	return &RuleCompiler{}
}

// Compile builds an event_type index for O(1) rule lookup per event.
// Invalid rule conditions are skipped and returned as non-fatal compile errors.
func (c *RuleCompiler) Compile(rules []storedRule) (map[string][]CompiledRule, []error) {
	indexed := make(map[string][]CompiledRule)
	compileErrs := make([]error, 0)

	for _, stored := range rules {
		compiled, err := c.compileOne(stored)
		if err != nil {
			compileErrs = append(compileErrs, fmt.Errorf("rule %s: %w", stored.ID, err))
			continue
		}

		for _, triggerType := range c.triggerEventTypes(compiled) {
			indexed[triggerType] = append(indexed[triggerType], compiled)
		}
	}

	return indexed, compileErrs
}

func (c *RuleCompiler) triggerEventTypes(rule CompiledRule) []string {
	// One rule can now subscribe to multiple trigger event types (sequence/correlation).
	switch rule.Type {
	case "threshold", "anomaly":
		return []string{rule.EventType}
	case "sequence":
		return dedupeStrings(rule.SequenceSteps)
	case "correlation":
		return dedupeStrings(rule.CorrelationEventTypes)
	default:
		return nil
	}
}

func (c *RuleCompiler) compileOne(rule storedRule) (CompiledRule, error) {
	var condition ruleCondition
	if err := json.Unmarshal(rule.Condition, &condition); err != nil {
		return CompiledRule{}, fmt.Errorf("invalid condition json: %w", err)
	}

	ruleType := strings.TrimSpace(strings.ToLower(condition.Type))
	if ruleType == "" {
		ruleType = "threshold"
	}

	if condition.WindowSec <= 0 {
		return CompiledRule{}, fmt.Errorf("window_sec must be > 0")
	}

	severity := strings.TrimSpace(condition.Severity)
	if severity == "" {
		severity = strings.TrimSpace(rule.Level)
	}
	if severity == "" {
		severity = "LOW"
	}

	groupBy := make([]string, 0, len(condition.GroupBy))
	for _, field := range condition.GroupBy {
		clean := strings.TrimSpace(strings.ToLower(field))
		if clean == "" {
			continue
		}
		groupBy = append(groupBy, clean)
	}

	compiled := CompiledRule{
		ID:       rule.ID,
		Type:     ruleType,
		Window:   time.Duration(condition.WindowSec) * time.Second,
		GroupBy:  groupBy,
		Severity: strings.ToUpper(severity),
	}

	switch ruleType {
	case "threshold":
		eventType := strings.TrimSpace(strings.ToLower(condition.EventType))
		if eventType == "" {
			return CompiledRule{}, fmt.Errorf("event_type is required")
		}
		if condition.Threshold <= 0 {
			return CompiledRule{}, fmt.Errorf("threshold must be > 0")
		}
		compiled.EventType = eventType
		compiled.Threshold = condition.Threshold
	case "sequence":
		if len(condition.SequenceSteps) < 2 {
			return CompiledRule{}, fmt.Errorf("sequence_steps must contain at least 2 event types")
		}
		steps := make([]string, 0, len(condition.SequenceSteps))
		for _, raw := range condition.SequenceSteps {
			step := strings.TrimSpace(strings.ToLower(raw))
			if step == "" {
				return CompiledRule{}, fmt.Errorf("sequence_steps must not contain empty values")
			}
			steps = append(steps, step)
		}
		compiled.SequenceSteps = steps
	case "correlation":
		if len(condition.CorrelationEventTypes) < 2 {
			return CompiledRule{}, fmt.Errorf("correlation_event_types must contain at least 2 event types")
		}
		eventTypes := make([]string, 0, len(condition.CorrelationEventTypes))
		for _, raw := range condition.CorrelationEventTypes {
			eventType := strings.TrimSpace(strings.ToLower(raw))
			if eventType == "" {
				return CompiledRule{}, fmt.Errorf("correlation_event_types must not contain empty values")
			}
			eventTypes = append(eventTypes, eventType)
		}
		if condition.MinDistinctEventTypes <= 0 {
			return CompiledRule{}, fmt.Errorf("min_distinct_event_types must be > 0")
		}
		if condition.MinDistinctEventTypes > len(eventTypes) {
			return CompiledRule{}, fmt.Errorf("min_distinct_event_types cannot exceed correlation_event_types length")
		}
		if condition.Threshold <= 0 {
			return CompiledRule{}, fmt.Errorf("threshold must be > 0")
		}
		compiled.CorrelationEventTypes = eventTypes
		compiled.MinDistinctEventTypes = condition.MinDistinctEventTypes
		compiled.Threshold = condition.Threshold
	case "anomaly":
		eventType := strings.TrimSpace(strings.ToLower(condition.EventType))
		if eventType == "" {
			return CompiledRule{}, fmt.Errorf("event_type is required")
		}
		if condition.BaselineWindowSec <= 0 {
			return CompiledRule{}, fmt.Errorf("baseline_window_sec must be > 0")
		}
		if condition.SpikeFactor <= 1 || math.IsNaN(condition.SpikeFactor) || math.IsInf(condition.SpikeFactor, 0) {
			return CompiledRule{}, fmt.Errorf("spike_factor must be > 1")
		}
		if condition.AnomalyMinCount <= 0 {
			return CompiledRule{}, fmt.Errorf("anomaly_min_count must be > 0")
		}
		compiled.EventType = eventType
		compiled.BaselineWindow = time.Duration(condition.BaselineWindowSec) * time.Second
		compiled.SpikeFactor = condition.SpikeFactor
		compiled.AnomalyMinCount = condition.AnomalyMinCount
	default:
		return CompiledRule{}, fmt.Errorf("unsupported type %q", ruleType)
	}

	return compiled, nil
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
