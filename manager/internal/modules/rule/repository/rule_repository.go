package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/rule/domain"
	"github.com/hildanku/xemarify/pkg/query"
)

// ListFilter holds filter and pagination options for listing rules.
type ListFilter struct {
	query.BaseFilter

	// Enabled filters by enabled status. nil means no filter.
	Enabled *bool

	// Level filters by severity level (INFO|LOW|MEDIUM|HIGH|CRITICAL).
	Level string
}

// RuleRepository defines the persistence contract for the rule module.
type RuleRepository interface {
	List(ctx context.Context, filter ListFilter) ([]*domain.Rule, int, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error)
	Create(ctx context.Context, rule *domain.Rule) error
	Update(ctx context.Context, rule *domain.Rule) error
	Delete(ctx context.Context, id uuid.UUID) error
}
