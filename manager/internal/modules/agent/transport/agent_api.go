package transport

// RegisterRequest is sent by an agent during first-time enrollment.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=255"`
	Hostname string `json:"hostname" binding:"required,min=1,max=255"`
	IP       string `json:"ip_address"`
	OS       string `json:"os"`
	Version  string `json:"version"`
}

// RegisterResponse contains the persistent agent identity returned after enrollment.
type RegisterResponse struct {
	AgentID string `json:"agent_id"`
	Key     string `json:"key"`
}

// HeartbeatRequest is sent periodically by authenticated agents.
type HeartbeatRequest struct {
	AgentID    string `json:"agent_id" binding:"required,uuid"`
	EventsSent int64  `json:"events_sent"`
	Uptime     int64  `json:"uptime"`
}

// CreateAgentKeyResponse is returned when admin generates one-time enrollment key.
type CreateAgentKeyResponse struct {
	Key string `json:"key"`
}
