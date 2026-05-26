package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/inventory/domain"
	inventoryRepo "github.com/hildanku/xemarify/internal/modules/inventory/repository"
	"github.com/sirupsen/logrus"
)

var ErrInventoryNotFound = errors.New("inventory not found")

// InventoryService handles business logic for agent inventory snapshots.
type InventoryService struct {
	repo inventoryRepo.InventoryRepository
	log  *logrus.Logger
}

// NewInventoryService constructs the service with its required dependencies.
func NewInventoryService(repo inventoryRepo.InventoryRepository, log *logrus.Logger) *InventoryService {
	return &InventoryService{repo: repo, log: log}
}

// UpsertInput carries the raw inventory payload sent by the agent.
type UpsertInput struct {
	AgentID         uuid.UUID
	OS              string
	Arch            string
	KernelVersion   string
	CPUModel        string
	CPUCores        int
	MemoryTotalMB   int64
	UptimeSeconds   int64
	IPAddresses     []string
	NginxInstalled  bool
	ApacheInstalled bool
	CollectedAt     time.Time
}

// Upsert stores or updates the inventory snapshot for the given agent.
func (s *InventoryService) Upsert(ctx context.Context, input UpsertInput) error {
	collectedAt := input.CollectedAt
	inv := &domain.Inventory{
		AgentID:         input.AgentID,
		OS:              input.OS,
		Arch:            input.Arch,
		KernelVersion:   input.KernelVersion,
		CPUModel:        input.CPUModel,
		CPUCores:        input.CPUCores,
		MemoryTotalMB:   input.MemoryTotalMB,
		UptimeSeconds:   input.UptimeSeconds,
		IPAddresses:     input.IPAddresses,
		NginxInstalled:  input.NginxInstalled,
		ApacheInstalled: input.ApacheInstalled,
		CollectedAt:     &collectedAt,
	}

	return s.repo.Upsert(ctx, inv)
}

// GetByAgentID returns the latest inventory snapshot for the given agent.
func (s *InventoryService) GetByAgentID(ctx context.Context, agentID uuid.UUID) (*domain.Inventory, error) {
	inv, err := s.repo.GetByAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, ErrInventoryNotFound
	}
	return inv, nil
}
