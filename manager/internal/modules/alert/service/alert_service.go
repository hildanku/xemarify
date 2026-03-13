package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/alert/domain"
	alertRepo "github.com/hildanku/xemarify/internal/modules/alert/repository"
	"github.com/sirupsen/logrus"
)

var (
	ErrAlertNotFound      = errors.New("alert not found")
	ErrInvalidAlertStatus = errors.New("invalid alert status")
)

type AlertService struct {
	repo alertRepo.AlertRepository
	log  *logrus.Logger
}

func NewAlertService(repo alertRepo.AlertRepository, log *logrus.Logger) *AlertService {
	return &AlertService{repo: repo, log: log}
}

func (s *AlertService) List(ctx context.Context, filter alertRepo.ListFilter) ([]*domain.Alert, int, error) {
	if filter.Status != "" {
		if err := validateStatus(filter.Status); err != nil {
			return nil, 0, err
		}
	}
	return s.repo.List(ctx, filter)
}

func (s *AlertService) GetByID(ctx context.Context, id uuid.UUID) (*domain.AlertDetail, error) {
	alert, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if alert == nil {
		return nil, ErrAlertNotFound
	}
	return alert, nil
}

func (s *AlertService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if err := validateStatus(status); err != nil {
		return err
	}

	updated, err := s.repo.UpdateStatus(ctx, id, strings.ToLower(strings.TrimSpace(status)))
	if err != nil {
		return err
	}
	if !updated {
		return ErrAlertNotFound
	}
	return nil
}

func validateStatus(status string) error {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case domain.AlertStatusNew, domain.AlertStatusAcknowledged, domain.AlertStatusClosed:
		return nil
	default:
		return fmt.Errorf("%w: must be one of new|acknowledged|closed", ErrInvalidAlertStatus)
	}
}
