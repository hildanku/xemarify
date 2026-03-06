package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/audit/domain"
	auditRepo "github.com/hildanku/xemarify/internal/modules/audit/repository"
	"github.com/sirupsen/logrus"
)

// AuditLogService handles writing and querying audit log entries.
type AuditLogService struct {
	repo auditRepo.AuditLogRepository
	log  *logrus.Logger
}

// NewAuditLogService constructs the service with its required dependencies.
func NewAuditLogService(repo auditRepo.AuditLogRepository, log *logrus.Logger) *AuditLogService {
	return &AuditLogService{repo: repo, log: log}
}

// Log writes an audit log entry. It never fails the caller — errors are only
// logged because audit logging should not interrupt the main request flow.
func (s *AuditLogService) Log(ctx context.Context, entry *domain.AuditLog) {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	if err := s.repo.Create(ctx, entry); err != nil {
		s.log.WithError(err).WithField("action", entry.Action).Error("failed to write audit log")
	}
}

// List returns paginated audit log entries matching the optional filters.
func (s *AuditLogService) List(ctx context.Context, filter auditRepo.ListFilter) ([]*domain.AuditLog, int, error) {
	return s.repo.List(ctx, filter)
}
