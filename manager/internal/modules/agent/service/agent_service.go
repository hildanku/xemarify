package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/agent/domain"
	agentRepo "github.com/hildanku/xemarify/internal/modules/agent/repository"
	"github.com/sirupsen/logrus"
)

var ErrAgentNotFound = errors.New("agent not found")

var ErrInvalidAgentStatus = errors.New("invalid agent status")

// AgentService handles business logic for the agent resource.
type AgentService struct {
	repo agentRepo.AgentRepository
	log  *logrus.Logger
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
