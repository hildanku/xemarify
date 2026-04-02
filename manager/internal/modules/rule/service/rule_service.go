package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/engine"
	"github.com/hildanku/xemarify/internal/modules/rule/domain"
	ruleRepo "github.com/hildanku/xemarify/internal/modules/rule/repository"
	"github.com/sirupsen/logrus"
)

var ErrInvalidRuleCondition = errors.New("invalid rule condition")

var validGroupByFields = map[string]struct{}{
	"src_ip":     {},
	"source_ip":  {},
	"ip":         {},
	"hostname":   {},
	"severity":   {},
	"category":   {},
	"facility":   {},
	"input_type": {},
	"agent_id":   {},
	"user":       {},
	"user_id":    {},
	"asset":      {},
	"asset_id":   {},
}

// RuleService orchestrates rule business logic.
type RuleService struct {
	repo   ruleRepo.RuleRepository
	engine engine.Engine
	log    *logrus.Logger
}

// NewRuleService constructs the service with its required dependencies.
func NewRuleService(repo ruleRepo.RuleRepository, detectionEngine engine.Engine, log *logrus.Logger) *RuleService {
	return &RuleService{repo: repo, engine: detectionEngine, log: log}
}

type CreateRuleInput struct {
	Name        string
	Description string
	Level       string
	Enabled     bool
	Condition   domain.RuleCondition
	Tags        []string
	CreatedByID *uuid.UUID
}

type UpdateRuleInput struct {
	Name        string
	Description string
	Level       string
	Enabled     *bool
	Condition   *domain.RuleCondition
	Tags        []string
}

func (s *RuleService) List(ctx context.Context, filter ruleRepo.ListFilter) ([]*domain.Rule, int, error) {
	return s.repo.List(ctx, filter)
}

func (s *RuleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *RuleService) Create(ctx context.Context, input CreateRuleInput) (*domain.Rule, error) {
	if err := validateCondition(input.Condition); err != nil {
		return nil, err
	}

	rule := &domain.Rule{
		ID:          uuid.New(),
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		Level:       strings.ToUpper(strings.TrimSpace(input.Level)),
		Enabled:     input.Enabled,
		Condition:   input.Condition,
		Tags:        input.Tags,
		Version:     1,
		CreatedBy:   input.CreatedByID,
	}
	if rule.Tags == nil {
		rule.Tags = []string{}
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		return nil, err
	}

	if s.engine != nil {
		if err := s.engine.ReloadRules(ctx); err != nil {
			return nil, fmt.Errorf("rule created but runtime reload failed: %w", err)
		}
	}

	return rule, nil
}

func (s *RuleService) Update(ctx context.Context, id uuid.UUID, input UpdateRuleInput) (*domain.Rule, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	if input.Name != "" {
		existing.Name = strings.TrimSpace(input.Name)
	}
	if input.Description != "" {
		existing.Description = strings.TrimSpace(input.Description)
	}
	if input.Level != "" {
		existing.Level = strings.ToUpper(strings.TrimSpace(input.Level))
	}
	if input.Enabled != nil {
		existing.Enabled = *input.Enabled
	}
	if input.Condition != nil {
		existing.Condition = *input.Condition
	}
	if input.Tags != nil {
		existing.Tags = input.Tags
	}

	if err := validateCondition(existing.Condition); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}

	if s.engine != nil {
		if err := s.engine.ReloadRules(ctx); err != nil {
			return nil, fmt.Errorf("rule updated but runtime reload failed: %w", err)
		}
	}

	return existing, nil
}

func (s *RuleService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	if s.engine != nil {
		if err := s.engine.ReloadRules(ctx); err != nil {
			return fmt.Errorf("rule deleted but runtime reload failed: %w", err)
		}
	}

	return nil
}

func validateCondition(condition domain.RuleCondition) error {
	ruleType := strings.ToLower(strings.TrimSpace(condition.Type))
	if ruleType == "" {
		ruleType = "threshold"
	}

	if condition.WindowSec <= 0 {
		return fmt.Errorf("%w: window_sec must be > 0", ErrInvalidRuleCondition)
	}

	switch ruleType {
	case "threshold":
		if strings.TrimSpace(condition.EventType) == "" {
			return fmt.Errorf("%w: event_type must not be empty", ErrInvalidRuleCondition)
		}
		if condition.Threshold <= 0 {
			return fmt.Errorf("%w: threshold must be > 0", ErrInvalidRuleCondition)
		}
	case "sequence":
		if len(condition.SequenceSteps) < 2 {
			return fmt.Errorf("%w: sequence_steps must contain at least 2 event types", ErrInvalidRuleCondition)
		}
		for _, step := range condition.SequenceSteps {
			if strings.TrimSpace(step) == "" {
				return fmt.Errorf("%w: sequence_steps must not contain empty values", ErrInvalidRuleCondition)
			}
		}
	case "correlation":
		if len(condition.CorrelationEventTypes) < 2 {
			return fmt.Errorf("%w: correlation_event_types must contain at least 2 event types", ErrInvalidRuleCondition)
		}
		for _, eventType := range condition.CorrelationEventTypes {
			if strings.TrimSpace(eventType) == "" {
				return fmt.Errorf("%w: correlation_event_types must not contain empty values", ErrInvalidRuleCondition)
			}
		}
		if condition.MinDistinctEventTypes <= 0 {
			return fmt.Errorf("%w: min_distinct_event_types must be > 0", ErrInvalidRuleCondition)
		}
		if condition.MinDistinctEventTypes > len(condition.CorrelationEventTypes) {
			return fmt.Errorf("%w: min_distinct_event_types cannot exceed correlation_event_types length", ErrInvalidRuleCondition)
		}
		if condition.Threshold <= 0 {
			return fmt.Errorf("%w: threshold must be > 0", ErrInvalidRuleCondition)
		}
	case "anomaly":
		if strings.TrimSpace(condition.EventType) == "" {
			return fmt.Errorf("%w: event_type must not be empty", ErrInvalidRuleCondition)
		}
		if condition.BaselineWindowSec <= 0 {
			return fmt.Errorf("%w: baseline_window_sec must be > 0", ErrInvalidRuleCondition)
		}
		if condition.SpikeFactor <= 1 || math.IsNaN(condition.SpikeFactor) || math.IsInf(condition.SpikeFactor, 0) {
			return fmt.Errorf("%w: spike_factor must be > 1", ErrInvalidRuleCondition)
		}
		if condition.AnomalyMinCount <= 0 {
			return fmt.Errorf("%w: anomaly_min_count must be > 0", ErrInvalidRuleCondition)
		}
	default:
		return fmt.Errorf("%w: unsupported type %q", ErrInvalidRuleCondition, condition.Type)
	}

	for _, field := range condition.GroupBy {
		normalized := strings.ToLower(strings.TrimSpace(field))
		if normalized == "" {
			continue
		}
		if _, ok := validGroupByFields[normalized]; !ok {
			return fmt.Errorf("%w: invalid group_by field %q", ErrInvalidRuleCondition, field)
		}
	}

	return nil
}
