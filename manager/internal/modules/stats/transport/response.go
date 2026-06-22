package transport

import statsRepo "github.com/hildanku/xemarify/internal/modules/stats/repository"

// DashboardStatsResponse is the payload for GET /api/v1/stats
type DashboardStatsResponse struct {
	Summary         *statsRepo.DashboardStats    `json:"summary"`
	ActivityTrend   []statsRepo.TrendPoint       `json:"activity_trend"`
	AlertStatusDist []statsRepo.AlertStatusCount `json:"alert_status_distribution"`
}
