package model

import "time"

const (
	AgentKeyHeader = "X-Agent-Key"
	AgentVersion   = "mvp-v1"
)

type RegisterRequest struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip_address"`
	OS       string `json:"os"`
	Version  string `json:"version"`
}

type RegisterResponse struct {
	AgentID string `json:"agent_id"`
	Key     string `json:"key"`
}

type HeartbeatRequest struct {
	AgentID    string `json:"agent_id"`
	EventsSent int64  `json:"events_sent"`
	Uptime     int64  `json:"uptime"`
}

type IngestEvent struct {
	EventTime  time.Time              `json:"event_time"`
	Hostname   string                 `json:"hostname"`
	SourceIP   string                 `json:"source_ip"`
	InputType  string                 `json:"input_type"`
	Facility   string                 `json:"facility"`
	Severity   string                 `json:"severity"`
	Category   string                 `json:"category"`
	Message    string                 `json:"message"`
	Normalized map[string]interface{} `json:"normalized"`
	Raw        string                 `json:"raw"`
}

type EventBatch struct {
	AgentID string        `json:"agent_id"`
	Events  []IngestEvent `json:"events"`
}
