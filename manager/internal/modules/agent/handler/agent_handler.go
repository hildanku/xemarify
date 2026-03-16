package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	rg.POST("", h.Create)
	rg.GET("/:id", h.GetByID)
	rg.PUT("/:id", h.Update)
	rg.DELETE("/:id", h.Delete)
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

	sortBy := q.SortBy
	if q.Sort != "" {
		sortBy = q.Sort
	}
	offset := q.Offset
	if offset == 0 && q.Page > 1 {
		offset = (q.Page - 1) * q.Limit
	}

	filter := agentRepo.ListFilter{
		BaseFilter: query.BaseFilter{
			Search: q.Search,
			SortBy: sortBy,
			Order:  query.SortOrder(q.Order),
			Limit:  q.Limit,
			Offset: offset,
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

func (h *AgentHandler) Create(c *gin.Context) {
	var req transport.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	a, err := h.svc.Create(c.Request.Context(), agentService.CreateAgentInput{
		Name:      req.Name,
		Hostname:  req.Hostname,
		IPAddress: req.IPAddress,
		Version:   req.Version,
		Status:    req.Status,
		Key:       req.Key,
	})
	if err != nil {
		if errors.Is(err, agentService.ErrInvalidAgentStatus) {
			response.Write(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		h.log.WithError(err).Error("failed to create agent")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusCreated, "agent created", transport.ToAgentResponse(a))
}

func (h *AgentHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid agent id", nil)
		return
	}

	a, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, agentService.ErrAgentNotFound) {
			response.Write(c, http.StatusNotFound, "agent not found", nil)
			return
		}
		h.log.WithError(err).Error("failed to get agent")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "agent retrieved", transport.ToAgentResponse(a))
}

func (h *AgentHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid agent id", nil)
		return
	}

	var req transport.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	a, err := h.svc.Update(c.Request.Context(), id, agentService.UpdateAgentInput{
		Name:      req.Name,
		Hostname:  req.Hostname,
		IPAddress: req.IPAddress,
		Version:   req.Version,
		Status:    req.Status,
	})
	if err != nil {
		if errors.Is(err, agentService.ErrAgentNotFound) {
			response.Write(c, http.StatusNotFound, "agent not found", nil)
			return
		}
		if errors.Is(err, agentService.ErrInvalidAgentStatus) {
			response.Write(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		h.log.WithError(err).Error("failed to update agent")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "agent updated", transport.ToAgentResponse(a))
}

func (h *AgentHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid agent id", nil)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, agentService.ErrAgentNotFound) {
			response.Write(c, http.StatusNotFound, "agent not found", nil)
			return
		}
		h.log.WithError(err).Error("failed to delete agent")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "agent deleted", nil)
}
