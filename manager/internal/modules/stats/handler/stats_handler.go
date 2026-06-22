package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	statsService "github.com/hildanku/xemarify/internal/modules/stats/service"
	"github.com/hildanku/xemarify/internal/modules/stats/transport"
	"github.com/hildanku/xemarify/pkg/response"
	"github.com/sirupsen/logrus"
)

type StatsHandler struct {
	svc *statsService.StatsService
	log *logrus.Logger
}

func NewStatsHandler(svc *statsService.StatsService, log *logrus.Logger) *StatsHandler {
	return &StatsHandler{svc: svc, log: log}
}

func (h *StatsHandler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.GetDashboardStats)
}

func (h *StatsHandler) GetDashboardStats(c *gin.Context) {
	ctx := c.Request.Context()

	summary, err := h.svc.GetDashboardStats(ctx)
	if err != nil {
		h.log.WithError(err).Error("failed to get dashboard stats")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	trend, err := h.svc.GetActivityTrend(ctx, 7)
	if err != nil {
		h.log.WithError(err).Error("failed to get activity trend")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	statusDistribution, err := h.svc.GetAlertStatusDistribution(ctx)
	if err != nil {
		h.log.WithError(err).Error("failed to get alert status distribution")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "stats retrieved", transport.DashboardStatsResponse{
		Summary:         summary,
		ActivityTrend:   trend,
		AlertStatusDist: statusDistribution,
	})
}
