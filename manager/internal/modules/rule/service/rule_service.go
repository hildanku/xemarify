package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/rule/domain"
	ruleRepo "github.com/hildanku/xemarify/internal/modules/rule/repository"
	"github.com/sirupsen/logrus"
)

// RuleService orchestrates rule business logic.
type RuleService struct {
	repo ruleRepo.RuleRepository
	log  *logrus.Logger
}

// NewRuleService constructs the service with its required dependencies.
func NewRuleService(repo ruleRepo.RuleRepository, log *logrus.Logger) *RuleService {
	return &RuleService{repo: repo, log: log}
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

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *RuleService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
