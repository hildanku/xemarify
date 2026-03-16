package transport

// ListAgentsQuery holds the query parameters for GET /api/v1/agents.
type ListAgentsQuery struct {
	// Search performs a case-insensitive partial match on name, hostname, and ip_address.
	Search string `form:"search"`

	// SortBy is the column to sort results by.
	// Allowed: name, hostname, status, created_at, last_seen_at, version.
	SortBy string `form:"sort_by,default=created_at"`
	Sort   string `form:"sort"`

	// Order is the sort direction: asc or desc.
	Order string `form:"order,default=asc" binding:"omitempty,oneof=asc desc"`

	// Page is 1-based page index. Used when offset is not provided.
	Page int `form:"page,default=1" binding:"omitempty,min=1"`

	// Limit is the maximum number of rows to return (1-100).
	Limit int `form:"limit,default=10" binding:"omitempty,min=1,max=100"`

	// Offset is the number of rows to skip.
	Offset int `form:"offset,default=0" binding:"omitempty,min=0"`
}

type CreateAgentRequest struct {
	Name      string `json:"name" binding:"required,min=1,max=255"`
	Hostname  string `json:"hostname"`
	IPAddress string `json:"ip_address"`
	Version   string `json:"version"`
	Status    string `json:"status" binding:"omitempty,oneof=ONLINE OFFLINE"`
	Key       string `json:"key"`
}

type UpdateAgentRequest struct {
	Name      string `json:"name" binding:"required,min=1,max=255"`
	Hostname  string `json:"hostname"`
	IPAddress string `json:"ip_address"`
	Version   string `json:"version"`
	Status    string `json:"status" binding:"required,oneof=ONLINE OFFLINE"`
}
