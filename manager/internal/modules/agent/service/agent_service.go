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
	"github.com/sirupsen/logrus"
)

var ErrAgentNotFound = errors.New("agent not found")

var ErrInvalidAgentStatus = errors.New("invalid agent status")

var ErrInvalidEnrollmentKey = errors.New("invalid enrollment key")

var ErrAgentIdentityMismatch = errors.New("agent identity mismatch")

// AgentService handles business logic for the agent resource.
type AgentService struct {
	repo            agentRepo.AgentRepository
	log             *logrus.Logger
	heartbeatStates sync.Map
}

// NewAgentService constructs the service with its required dependencies.
func NewAgentService(repo agentRepo.AgentRepository, log *logrus.Logger) *AgentService {
	return &AgentService{repo: repo, log: log}
}

// List returns a filtered, sorted, paginated list of agents and the total match count.
func (s *AgentService) List(ctx context.Context, filter agentRepo.ListFilter) ([]*domain.Agent, int, error) {
	return s.repo.List(ctx, filter)
}

type CreateAgentInput struct {
	Name      string
	Hostname  string
	IPAddress string
	Version   string
	Status    string
	Key       string
}

type RegisterInput struct {
	Name          string
	Hostname      string
	IPAddress     string
	OS            string
	Version       string
	EnrollmentKey string
}

type RegisterResult struct {
	AgentID string
	Key     string
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

func (s *AgentService) Create(ctx context.Context, input CreateAgentInput) (*domain.Agent, error) {
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
		Key:       strings.TrimSpace(input.Key),
	}

	if agent.Key == "" {
		agent.Key = uuid.NewString()
	}

	if err := s.repo.Create(ctx, agent); err != nil {
		return nil, err
	}

	return agent, nil
}

func (s *AgentService) Register(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	name := strings.TrimSpace(input.Name)
	hostname := strings.TrimSpace(input.Hostname)
	ipAddress := strings.TrimSpace(input.IPAddress)
	version := strings.TrimSpace(input.Version)
	enrollmentKey := strings.TrimSpace(input.EnrollmentKey)

	if name == "" {
		name = hostname
	}
	if hostname == "" {
		hostname = name
	}

	sessionKey, err := generateSessionKey()
	if err != nil {
		return nil, err
	}

	agent := &domain.Agent{
		ID:        uuid.New(),
		Name:      name,
		Hostname:  hostname,
		IPAddress: ipAddress,
		Version:   version,
		Status:    domain.AgentStatusOffline,
		Key:       sessionKey,
	}

	if err := s.repo.CreateWithEnrollmentKey(ctx, enrollmentKey, agent); err != nil {
		if errors.Is(err, agentRepo.ErrEnrollmentKeyInvalid) {
			return nil, ErrInvalidEnrollmentKey
		}
		return nil, err
	}

	return &RegisterResult{
		AgentID: agent.ID.String(),
		Key:     sessionKey,
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

func (s *AgentService) GenerateEnrollmentKey(ctx context.Context) (string, error) {
	key, err := generateSessionKey()
	if err != nil {
		return "", err
	}

	if err := s.repo.CreateEnrollmentKey(ctx, key); err != nil {
		return "", err
	}

	return key, nil
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

func (s *AgentService) Update(ctx context.Context, id uuid.UUID, input UpdateAgentInput) (*domain.Agent, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAgentNotFound
	}

	status, err := normalizeStatus(input.Status)
	if err != nil {
		return nil, err
	}

	a.Name = strings.TrimSpace(input.Name)
	a.Hostname = strings.TrimSpace(input.Hostname)
	a.IPAddress = strings.TrimSpace(input.IPAddress)
	a.Version = strings.TrimSpace(input.Version)
	a.Status = status

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

	return updated, nil
}

func (s *AgentService) Delete(ctx context.Context, id uuid.UUID) error {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrAgentNotFound
	}

	return s.repo.Delete(ctx, id)
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
