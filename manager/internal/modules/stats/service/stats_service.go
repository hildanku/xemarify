package service

import (
	"context"

	statsRepo "github.com/hildanku/xemarify/internal/modules/stats/repository"
	"github.com/sirupsen/logrus"
)

type StatsService struct {
	repo statsRepo.StatsRepository
	log  *logrus.Logger
}

func NewStatsService(repo statsRepo.StatsRepository, log *logrus.Logger) *StatsService {
	return &StatsService{repo: repo, log: log}
}

func (s *StatsService) GetDashboardStats(ctx context.Context) (*statsRepo.DashboardStats, error) {
	return s.repo.GetDashboardStats(ctx)
}

func (s *StatsService) GetActivityTrend(ctx context.Context, days int) ([]statsRepo.TrendPoint, error) {
	if days <= 0 || days > 90 {
		days = 7
	}
	return s.repo.GetActivityTrend(ctx, days)
}

func (s *StatsService) GetAlertStatusDistribution(ctx context.Context) ([]statsRepo.AlertStatusCount, error) {
	return s.repo.GetAlertStatusDistribution(ctx)
}
