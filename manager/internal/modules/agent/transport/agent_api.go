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
	AgentID     string `json:"agent_id"`
	AgentSecret string `json:"agent_secret"`
}

// HeartbeatRequest is sent periodically by authenticated agents.
type HeartbeatRequest struct {
	AgentID    string `json:"agent_id" binding:"required,uuid"`
	EventsSent int64  `json:"events_sent"`
	Uptime     int64  `json:"uptime"`
}

// CreateEnrollmentTokenResponse is returned when admin generates a one-time enrollment token.
type CreateEnrollmentTokenResponse struct {
	EnrollmentToken string `json:"enrollment_token"`
}
