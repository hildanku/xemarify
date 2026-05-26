package domain

import (
	"time"

	"github.com/google/uuid"
)

// Inventory holds the latest system snapshot for a single agent.
type Inventory struct {
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
	CollectedAt     *time.Time
	UpdatedAt       time.Time
}
