package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/alert/domain"
	"github.com/hildanku/xemarify/pkg/query"
)

type ListFilter struct {
	query.BaseFilter
	Severity      string
	Status        string
	RuleID        *uuid.UUID
	TriggeredFrom *time.Time
	TriggeredTo   *time.Time
}

type AlertRepository interface {
	List(ctx context.Context, filter ListFilter) ([]*domain.Alert, int, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AlertDetail, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (bool, error)
}
