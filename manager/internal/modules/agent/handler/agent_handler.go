package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	agentRepo "github.com/hildanku/xemarify/internal/modules/agent/repository"
	agentService "github.com/hildanku/xemarify/internal/modules/agent/service"
	"github.com/hildanku/xemarify/internal/modules/agent/transport"
	"github.com/hildanku/xemarify/pkg/query"
	"github.com/hildanku/xemarify/pkg/response"
	"github.com/sirupsen/logrus"
)

// AgentHandler handles HTTP requests for the agent management endpoints.
type AgentHandler struct {
	svc *agentService.AgentService
	log *logrus.Logger
}

// NewAgentHandler constructs an AgentHandler.
func NewAgentHandler(svc *agentService.AgentService, log *logrus.Logger) *AgentHandler {
	return &AgentHandler{svc: svc, log: log}
}

// Register wires the agent management routes onto the given router group.
// The group must already have JWT + RBAC middleware applied.
func (h *AgentHandler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
}

// List handles GET /api/v1/agents.
//
// Query params:
//
//	search    - case-insensitive partial match on name, hostname, ip_address
//	sort_by   - field to sort by (name|hostname|status|created_at|last_seen_at|version); default: created_at
//	order     - sort direction (asc|desc); default: asc
//	limit     - max rows to return (1-100); default: 10
//	offset    - rows to skip; default: 0
func (h *AgentHandler) List(c *gin.Context) {
	var q transport.ListAgentsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	filter := agentRepo.ListFilter{
		BaseFilter: query.BaseFilter{
			Search: q.Search,
			SortBy: q.SortBy,
			Order:  query.SortOrder(q.Order),
			Limit:  q.Limit,
			Offset: q.Offset,
		},
	}

	agents, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.log.WithError(err).Error("failed to list agents")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	items := make([]*transport.AgentResponse, 0, len(agents))
	for _, a := range agents {
		items = append(items, transport.ToAgentResponse(a))
	}

	totalPages := 0
	if filter.Limit > 0 {
		totalPages = (total + filter.Limit - 1) / filter.Limit
	}

	response.Write(c, http.StatusOK, "agents retrieved", transport.ListAgentsResponse{
		Items: items,
		Metadata: transport.ListAgentsMetadata{
			Total:      total,
			TotalPages: totalPages,
			Limit:      filter.Limit,
			Offset:     filter.Offset,
		},
	})
}
