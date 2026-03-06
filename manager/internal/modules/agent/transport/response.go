package transport

import (
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/agent/domain"
)

// AgentResponse is the JSON representation of a single agent.
type AgentResponse struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Hostname   string     `json:"hostname"`
	IPAddress  string     `json:"ip_address"`
	Version    string     `json:"version"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
}

// ListAgentsMetadata carries pagination and count information for a list response.
type ListAgentsMetadata struct {
	// Total is the total number of agents matching the current filter (ignores limit/offset).
	Total int `json:"total"`

	// TotalPages is the total number of pages given the current limit.
	TotalPages int `json:"total_pages"`

	// Limit is the page size that was applied.
	Limit int `json:"limit"`

	// Offset is the number of rows skipped.
	Offset int `json:"offset"`
}

// ListAgentsResponse wraps a slice of agents together with pagination metadata.
type ListAgentsResponse struct {
	Items    []*AgentResponse   `json:"items"`
	Metadata ListAgentsMetadata `json:"metadata"`
}

// ToAgentResponse converts a domain Agent to its HTTP response form.
func ToAgentResponse(a *domain.Agent) *AgentResponse {
	return &AgentResponse{
		ID:         a.ID,
		Name:       a.Name,
		Hostname:   a.Hostname,
		IPAddress:  a.IPAddress,
		Version:    a.Version,
		Status:     string(a.Status),
		CreatedAt:  a.CreatedAt,
		LastSeenAt: a.LastSeenAt,
	}
}
