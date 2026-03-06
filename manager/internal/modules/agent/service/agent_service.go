package service

import (
	"context"

	"github.com/hildanku/xemarify/internal/modules/agent/domain"
	agentRepo "github.com/hildanku/xemarify/internal/modules/agent/repository"
	"github.com/sirupsen/logrus"
)

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
