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

// ListAgentsMetadata carries pagination info for a list response.
// COUNT(*) and total_pages have been removed in favour of keyset pagination.
type ListAgentsMetadata struct {
	// NextCursor is the opaque token to pass as ?cursor= on the next request.
	// An empty string means this is the last page.
	NextCursor string `json:"next_cursor"`

	// HasMore is a convenience boolean derived from NextCursor.
	HasMore bool `json:"has_more"`

	// Limit is the page size that was applied.
	Limit int `json:"limit"`
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