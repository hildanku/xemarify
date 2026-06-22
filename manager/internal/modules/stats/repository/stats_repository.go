package repository

import "context"

// DashboardStats holds aggregate counts for the dashboard.
type DashboardStats struct {
	TotalEvents   int64 `json:"total_events"`
	TotalAlerts   int64 `json:"total_alerts"`
	NewAlerts     int64 `json:"new_alerts"`
	TotalAgents   int64 `json:"total_agents"`
	OnlineAgents  int64 `json:"online_agents"`
	AuditLogTotal int64 `json:"audit_log_total"`
}

// TrendPoint holds event and alert counts for a single day.
type TrendPoint struct {
	Day    string `json:"day"`
	Events int64  `json:"events"`
	Alerts int64  `json:"alerts"`
}

// AlertStatusCount holds count per alert status.
type AlertStatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// StatsRepository defines the persistence contract for dashboard stats.
type StatsRepository interface {
	// GetDashboardStats returns aggregate counts for the dashboard summary cards.
	GetDashboardStats(ctx context.Context) (*DashboardStats, error)

	// GetActivityTrend returns per-day event and alert counts for the last N days.
	GetActivityTrend(ctx context.Context, days int) ([]TrendPoint, error)

	// GetAlertStatusDistribution returns alert counts grouped by status.
	GetAlertStatusDistribution(ctx context.Context) ([]AlertStatusCount, error)
}
