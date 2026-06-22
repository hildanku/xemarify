package transport

// ListAgentsQuery holds the query parameters for GET /api/v1/agents.
type ListAgentsQuery struct {
	// Search performs a case-insensitive partial match on name, hostname, and ip_address.
	Search string `form:"search"`

	// Order is the sort direction: asc or desc.
	Order string `form:"order,default=desc" binding:"omitempty,oneof=asc desc"`

	// Limit is the maximum number of rows to return (1-100).
	Limit int `form:"limit,default=10" binding:"omitempty,min=1,max=100"`

	// Cursor is the opaque pagination token from the previous response's
	// next_cursor field. Omit or leave empty to fetch the first page.
	Cursor string `form:"cursor"`
}

type CreateAgentRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Hostname    string `json:"hostname"`
	IPAddress   string `json:"ip_address"`
	Version     string `json:"version"`
	Status      string `json:"status" binding:"omitempty,oneof=ONLINE OFFLINE"`
	AgentSecret string `json:"agent_secret"`
}

type UpdateAgentRequest struct {
	Name      string `json:"name" binding:"required,min=1,max=255"`
	Hostname  string `json:"hostname"`
	IPAddress string `json:"ip_address"`
	Version   string `json:"version"`
	Status    string `json:"status" binding:"required,oneof=ONLINE OFFLINE"`
}