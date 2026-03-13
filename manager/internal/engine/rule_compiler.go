package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CompiledRule is the runtime representation of a detection rule.
// Rules are precompiled at startup and indexed by event type for fast lookups.
type CompiledRule struct {
	ID        uuid.UUID
	EventType string
	Threshold int
	Window    time.Duration
	GroupBy   []string
	Severity  string
}

type storedRule struct {
	ID        uuid.UUID
	Level     string
	Condition []byte
}

type ruleCondition struct {
	EventType string   `json:"event_type"`
	GroupBy   []string `json:"group_by"`
	Threshold int      `json:"threshold"`
	WindowSec int      `json:"window_sec"`
	Severity  string   `json:"severity"`
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

		indexed[compiled.EventType] = append(indexed[compiled.EventType], compiled)
	}

	return indexed, compileErrs
}

func (c *RuleCompiler) compileOne(rule storedRule) (CompiledRule, error) {
	var condition ruleCondition
	if err := json.Unmarshal(rule.Condition, &condition); err != nil {
		return CompiledRule{}, fmt.Errorf("invalid condition json: %w", err)
	}

	eventType := strings.TrimSpace(strings.ToLower(condition.EventType))
	if eventType == "" {
		return CompiledRule{}, fmt.Errorf("event_type is required")
	}
	if condition.Threshold <= 0 {
		return CompiledRule{}, fmt.Errorf("threshold must be > 0")
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

	return CompiledRule{
		ID:        rule.ID,
		EventType: eventType,
		Threshold: condition.Threshold,
		Window:    time.Duration(condition.WindowSec) * time.Second,
		GroupBy:   groupBy,
		Severity:  strings.ToUpper(severity),
	}, nil
}
