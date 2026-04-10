package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/alert/domain"
	alertRepo "github.com/hildanku/xemarify/internal/modules/alert/repository"
	auditDomain "github.com/hildanku/xemarify/internal/modules/audit/domain"
	auditService "github.com/hildanku/xemarify/internal/modules/audit/service"
	jwtpkg "github.com/hildanku/xemarify/pkg/jwt"
	"github.com/sirupsen/logrus"
)

var (
	ErrAlertNotFound      = errors.New("alert not found")
	ErrInvalidAlertStatus = errors.New("invalid alert status")
)

type AlertService struct {
	repo     alertRepo.AlertRepository
	auditSvc *auditService.AuditLogService
	log      *logrus.Logger
}

func NewAlertService(repo alertRepo.AlertRepository, auditSvc *auditService.AuditLogService, log *logrus.Logger) *AlertService {
	return &AlertService{repo: repo, auditSvc: auditSvc, log: log}
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

func (s *AlertService) UpdateStatus(ctx context.Context, id uuid.UUID, status string, actor *jwtpkg.Claims, ip string) error {
	if err := validateStatus(status); err != nil {
		return err
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrAlertNotFound
	}

	normalizedStatus := strings.ToLower(strings.TrimSpace(status))

	updated, err := s.repo.UpdateStatus(ctx, id, normalizedStatus)
	if err != nil {
		return err
	}
	if !updated {
		return ErrAlertNotFound
	}

	s.auditSvc.Log(ctx, &auditDomain.AuditLog{
		UserID:         &actor.UserID,
		UserIdentifier: actor.Username,
		Action:         auditDomain.ActionUpdateAlertStatus,
		ObjectType:     strPtr(auditDomain.ObjectTypeAlert),
		ObjectID:       &id,
		Metadata: map[string]interface{}{
			"old_status": existing.Alert.Status,
			"new_status": normalizedStatus,
			"rule_name":  existing.Alert.RuleName,
			"ip_address": ip,
		},
	})
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

func strPtr(s string) *string { return &s }
