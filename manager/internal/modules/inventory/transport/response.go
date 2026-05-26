package transport

import (
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/inventory/domain"
)

// InventoryRequest is the JSON body sent by the agent to POST /api/v1/agents/inventory.
type InventoryRequest struct {
	AgentID         string    `json:"agent_id"`
	OS              string    `json:"os"`
	Arch            string    `json:"arch"`
	KernelVersion   string    `json:"kernel_version"`
	CPUModel        string    `json:"cpu_model"`
	CPUCores        int       `json:"cpu_cores"`
	MemoryTotalMB   int64     `json:"memory_total_mb"`
	UptimeSeconds   int64     `json:"uptime_seconds"`
	IPAddresses     []string  `json:"ip_addresses"`
	NginxInstalled  bool      `json:"nginx_installed"`
	ApacheInstalled bool      `json:"apache_installed"`
	CollectedAt     time.Time `json:"collected_at"`
}

// InventoryResponse is the JSON body returned by GET /api/v1/agents/:id/inventory.
type InventoryResponse struct {
	AgentID         uuid.UUID  `json:"agent_id"`
	OS              string     `json:"os"`
	Arch            string     `json:"arch"`
	KernelVersion   string     `json:"kernel_version"`
	CPUModel        string     `json:"cpu_model"`
	CPUCores        int        `json:"cpu_cores"`
	MemoryTotalMB   int64      `json:"memory_total_mb"`
	UptimeSeconds   int64      `json:"uptime_seconds"`
	IPAddresses     []string   `json:"ip_addresses"`
	NginxInstalled  bool       `json:"nginx_installed"`
	ApacheInstalled bool       `json:"apache_installed"`
	CollectedAt     *time.Time `json:"collected_at,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ToInventoryResponse converts a domain Inventory to its HTTP response form.
func ToInventoryResponse(inv *domain.Inventory) *InventoryResponse {
	ips := inv.IPAddresses
	if ips == nil {
		ips = []string{}
	}
	return &InventoryResponse{
		AgentID:         inv.AgentID,
		OS:              inv.OS,
		Arch:            inv.Arch,
		KernelVersion:   inv.KernelVersion,
		CPUModel:        inv.CPUModel,
		CPUCores:        inv.CPUCores,
		MemoryTotalMB:   inv.MemoryTotalMB,
		UptimeSeconds:   inv.UptimeSeconds,
		IPAddresses:     ips,
		NginxInstalled:  inv.NginxInstalled,
		ApacheInstalled: inv.ApacheInstalled,
		CollectedAt:     inv.CollectedAt,
		UpdatedAt:       inv.UpdatedAt,
	}
}
