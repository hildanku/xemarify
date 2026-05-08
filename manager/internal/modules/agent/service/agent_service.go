package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/agent/domain"
	agentRepo "github.com/hildanku/xemarify/internal/modules/agent/repository"
	auditDomain "github.com/hildanku/xemarify/internal/modules/audit/domain"
	auditService "github.com/hildanku/xemarify/internal/modules/audit/service"
	jwtpkg "github.com/hildanku/xemarify/pkg/jwt"
	"github.com/sirupsen/logrus"
)

var ErrAgentNotFound = errors.New("agent not found")

var ErrInvalidAgentStatus = errors.New("invalid agent status")

var ErrInvalidEnrollmentToken = errors.New("invalid enrollment token")

var ErrAgentIdentityMismatch = errors.New("agent identity mismatch")

// AgentService handles business logic for the agent resource.
type AgentService struct {
	repo            agentRepo.AgentRepository
	auditSvc        *auditService.AuditLogService
	log             *logrus.Logger
	heartbeatStates sync.Map
}

// NewAgentService constructs the service with its required dependencies.
func NewAgentService(repo agentRepo.AgentRepository, auditSvc *auditService.AuditLogService, log *logrus.Logger) *AgentService {
	return &AgentService{repo: repo, auditSvc: auditSvc, log: log}
}

// List returns a filtered, sorted, paginated list of agents and the total match count.
func (s *AgentService) List(ctx context.Context, filter agentRepo.ListFilter) ([]*domain.Agent, int, error) {
	return s.repo.List(ctx, filter)
}

type CreateAgentInput struct {
	Name        string
	Hostname    string
	IPAddress   string
	Version     string
	Status      string
	AgentSecret string
}

type RegisterInput struct {
	Name            string
	Hostname        string
	IPAddress       string
	OS              string
	Version         string
	EnrollmentToken string
}

type RegisterResult struct {
	AgentID     string
	AgentSecret string
}

type HeartbeatInput struct {
	AuthenticatedAgentID uuid.UUID
	AgentID              string
	EventsSent           int64
	Uptime               int64
}

type heartbeatState struct {
	EventsSent int64
	Uptime     int64
	UpdatedAt  time.Time
}

func (s *AgentService) Create(ctx context.Context, input CreateAgentInput, actor *jwtpkg.Claims, ip string) (*domain.Agent, error) {
	status, err := normalizeStatus(input.Status)
	if err != nil {
		return nil, err
	}

	agent := &domain.Agent{
		ID:        uuid.New(),
		Name:      strings.TrimSpace(input.Name),
		Hostname:  strings.TrimSpace(input.Hostname),
		IPAddress: strings.TrimSpace(input.IPAddress),
		Version:   strings.TrimSpace(input.Version),
		Status:    status,
		Secret:    strings.TrimSpace(input.AgentSecret),
	}

	if agent.Secret == "" {
		agent.Secret = uuid.NewString()
	}

	if err := s.repo.Create(ctx, agent); err != nil {
		return nil, err
	}

	s.auditSvc.Log(ctx, &auditDomain.AuditLog{
		UserID:         &actor.UserID,
		UserIdentifier: actor.Username,
		Action:         auditDomain.ActionCreateAgent,
		ObjectType:     strPtr(auditDomain.ObjectTypeAgent),
		ObjectID:       &agent.ID,
		Metadata: map[string]interface{}{
			"agent_name": agent.Name,
			"hostname":   agent.Hostname,
			"status":     agent.Status,
			"ip_address": ip,
		},
	})

	return agent, nil
}

func (s *AgentService) Register(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	name := strings.TrimSpace(input.Name)
	hostname := strings.TrimSpace(input.Hostname)
	ipAddress := strings.TrimSpace(input.IPAddress)
	version := strings.TrimSpace(input.Version)
	enrollmentToken := strings.TrimSpace(input.EnrollmentToken)

	if name == "" {
		name = hostname
	}
	if hostname == "" {
		hostname = name
	}

	agentSecret, err := generateSessionKey()
	if err != nil {
		return nil, err
	}

	agent := &domain.Agent{
		ID:        uuid.New(),
		Name:      name,
		Hostname:  hostname,
		IPAddress: ipAddress,
		Version:   version,
		Status:    domain.AgentStatusOnline,
		Secret:    agentSecret,
	}

	if err := s.repo.CreateWithEnrollmentToken(ctx, enrollmentToken, agent); err != nil {
		if errors.Is(err, agentRepo.ErrEnrollmentTokenInvalid) {
			return nil, ErrInvalidEnrollmentToken
		}
		return nil, err
	}

	s.auditSvc.Log(ctx, &auditDomain.AuditLog{
		Action:         auditDomain.ActionRegisterAgent,
		UserIdentifier: agent.Name,
		ObjectType:     strPtr(auditDomain.ObjectTypeAgent),
		ObjectID:       &agent.ID,
		Metadata: map[string]interface{}{
			"agent_name": agent.Name,
			"hostname":   agent.Hostname,
			"ip_address": agent.IPAddress,
			"version":    agent.Version,
		},
	})

	return &RegisterResult{
		AgentID:     agent.ID.String(),
		AgentSecret: agentSecret,
	}, nil
}

func (s *AgentService) Heartbeat(ctx context.Context, input HeartbeatInput) error {
	heartbeatAgentID, err := uuid.Parse(strings.TrimSpace(input.AgentID))
	if err != nil {
		return fmt.Errorf("invalid agent id: %w", err)
	}

	if heartbeatAgentID != input.AuthenticatedAgentID {
		return ErrAgentIdentityMismatch
	}

	a, err := s.repo.GetByID(ctx, heartbeatAgentID)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrAgentNotFound
	}

	if err := s.repo.UpdateLastSeen(ctx, heartbeatAgentID); err != nil {
		return err
	}

	s.heartbeatStates.Store(heartbeatAgentID, heartbeatState{
		EventsSent: input.EventsSent,
		Uptime:     input.Uptime,
		UpdatedAt:  time.Now().UTC(),
	})

	return nil
}

func (s *AgentService) GenerateEnrollmentToken(ctx context.Context, actor *jwtpkg.Claims, ip string) (string, error) {
	token, err := generateSessionKey()
	if err != nil {
		return "", err
	}

	if err := s.repo.CreateEnrollmentToken(ctx, token); err != nil {
		return "", err
	}

	s.auditSvc.Log(ctx, &auditDomain.AuditLog{
		UserID:         &actor.UserID,
		UserIdentifier: actor.Username,
		Action:         auditDomain.ActionGenerateEnrollmentToken,
		ObjectType:     strPtr(auditDomain.ObjectTypeEnrollmentToken),
		Metadata: map[string]interface{}{
			"ip_address": ip,
		},
	})

	return token, nil
}

func (s *AgentService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAgentNotFound
	}
	return a, nil
}

type UpdateAgentInput struct {
	Name      string
	Hostname  string
	IPAddress string
	Version   string
	Status    string
}

func (s *AgentService) Update(ctx context.Context, id uuid.UUID, input UpdateAgentInput, actor *jwtpkg.Claims, ip string) (*domain.Agent, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAgentNotFound
	}

	var changedFields []string

	status, err := normalizeStatus(input.Status)
	if err != nil {
		return nil, err
	}

	if trimmed := strings.TrimSpace(input.Name); trimmed != a.Name {
		changedFields = append(changedFields, "name")
		a.Name = trimmed
	}
	if trimmed := strings.TrimSpace(input.Hostname); trimmed != a.Hostname {
		changedFields = append(changedFields, "hostname")
		a.Hostname = trimmed
	}
	if trimmed := strings.TrimSpace(input.IPAddress); trimmed != a.IPAddress {
		changedFields = append(changedFields, "ip_address")
		a.IPAddress = trimmed
	}
	if trimmed := strings.TrimSpace(input.Version); trimmed != a.Version {
		changedFields = append(changedFields, "version")
		a.Version = trimmed
	}
	if status != a.Status {
		changedFields = append(changedFields, "status")
		a.Status = status
	}

	if err := s.repo.Update(ctx, id, a); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrAgentNotFound
	}

	s.auditSvc.Log(ctx, &auditDomain.AuditLog{
		UserID:         &actor.UserID,
		UserIdentifier: actor.Username,
		Action:         auditDomain.ActionUpdateAgent,
		ObjectType:     strPtr(auditDomain.ObjectTypeAgent),
		ObjectID:       &id,
		Metadata: map[string]interface{}{
			"changed_fields": changedFields,
			"ip_address":     ip,
		},
	})

	return updated, nil
}

func (s *AgentService) Delete(ctx context.Context, id uuid.UUID, actor *jwtpkg.Claims, ip string) error {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrAgentNotFound
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	s.auditSvc.Log(ctx, &auditDomain.AuditLog{
		UserID:         &actor.UserID,
		UserIdentifier: actor.Username,
		Action:         auditDomain.ActionDeleteAgent,
		ObjectType:     strPtr(auditDomain.ObjectTypeAgent),
		ObjectID:       &id,
		Metadata: map[string]interface{}{
			"deleted_agent_name": a.Name,
			"hostname":           a.Hostname,
			"ip_address":         ip,
		},
	})

	return nil
}

func normalizeStatus(status string) (domain.AgentStatus, error) {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	if normalized == "" {
		return domain.AgentStatusOffline, nil
	}

	switch domain.AgentStatus(normalized) {
	case domain.AgentStatusOnline, domain.AgentStatusOffline:
		return domain.AgentStatus(normalized), nil
	default:
		return "", ErrInvalidAgentStatus
	}
}

func generateSessionKey() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return hex.EncodeToString(raw), nil
}

func strPtr(s string) *string { return &s }
