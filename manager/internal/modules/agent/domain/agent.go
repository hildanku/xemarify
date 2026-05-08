package domain

import (
	"time"

	"github.com/google/uuid"
)

type AgentStatus string

const (
	AgentStatusOnline  AgentStatus = "ONLINE"
	AgentStatusOffline AgentStatus = "OFFLINE"
)

// Agent is the internal domain representation of a registered agent.
type Agent struct {
	ID         uuid.UUID
	Name       string
	Hostname   string
	Secret     string
	IPAddress  string
	Version    string
	Status     AgentStatus
	CreatedAt  time.Time
	LastSeenAt *time.Time
}
