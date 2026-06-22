package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/infrastructure/middleware"
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

const enrollmentTokenHeader = "X-Enrollment-Token"

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

// RegisterAgentPublic wires public agent enrollment routes.
func (h *AgentHandler) RegisterAgentPublic(rg *gin.RouterGroup) {
	rg.POST("/register", h.RegisterAgent)
}

// RegisterAgentSession wires authenticated agent routes.
func (h *AgentHandler) RegisterAgentSession(rg *gin.RouterGroup) {
	rg.POST("/heartbeat", h.Heartbeat)
}

// RegisterAdmin wires manager-only admin routes under /api/v1/admin.
func (h *AgentHandler) RegisterAdmin(rg *gin.RouterGroup) {
	rg.POST("/enrollment-tokens", h.CreateEnrollmentToken)
}

// RegisterAgent handles POST /api/v1/agents/register.
func (h *AgentHandler) RegisterAgent(c *gin.Context) {
	enrollmentToken := strings.TrimSpace(c.GetHeader(enrollmentTokenHeader))
	if enrollmentToken == "" {
		response.Write(c, http.StatusUnauthorized, "missing X-Enrollment-Token header", nil)
		return
	}

	var req transport.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	registered, err := h.svc.Register(c.Request.Context(), agentService.RegisterInput{
		Name:            req.Name,
		Hostname:        req.Hostname,
		IPAddress:       req.IP,
		OS:              req.OS,
		Version:         req.Version,
		EnrollmentToken: enrollmentToken,
	})
	if err != nil {
		if errors.Is(err, agentService.ErrInvalidEnrollmentToken) {
			response.Write(c, http.StatusUnauthorized, "invalid enrollment token", nil)
			return
		}

		h.log.WithError(err).Error("failed to register agent")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	c.JSON(http.StatusCreated, transport.RegisterResponse{
		AgentID:     registered.AgentID,
		AgentSecret: registered.AgentSecret,
	})
}

// Heartbeat handles POST /api/v1/agents/heartbeat.
func (h *AgentHandler) Heartbeat(c *gin.Context) {
	authenticatedAgent := middleware.AgentFromContext(c)
	if authenticatedAgent == nil {
		response.Write(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req transport.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err := h.svc.Heartbeat(c.Request.Context(), agentService.HeartbeatInput{
		AuthenticatedAgentID: authenticatedAgent.ID,
		AgentID:              req.AgentID,
		EventsSent:           req.EventsSent,
		Uptime:               req.Uptime,
	})
	if err != nil {
		if errors.Is(err, agentService.ErrAgentIdentityMismatch) {
			response.Write(c, http.StatusForbidden, "agent identity mismatch", nil)
			return
		}
		if errors.Is(err, agentService.ErrAgentNotFound) {
			response.Write(c, http.StatusNotFound, "agent not found", nil)
			return
		}

		h.log.WithError(err).Error("failed to process heartbeat")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "heartbeat accepted", nil)
}

// CreateEnrollmentToken handles POST /api/v1/admin/enrollment-tokens.
func (h *AgentHandler) CreateEnrollmentToken(c *gin.Context) {
	claims := middleware.UserClaimsFromContext(c)

	token, err := h.svc.GenerateEnrollmentToken(c.Request.Context(), claims, c.ClientIP())
	if err != nil {
		h.log.WithError(err).Error("failed to create enrollment token")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	c.JSON(http.StatusCreated, transport.CreateEnrollmentTokenResponse{EnrollmentToken: token})
}

// List handles GET /api/v1/agents.
//
// Query params:
//
//	search    - case-insensitive partial match on name, hostname, ip_address
//	order     - sort direction (asc|desc); default: desc
//	limit     - max rows to return (1-100); default: 10
//	cursor    - opaque keyset pagination token from previous response's next_cursor field
func (h *AgentHandler) List(c *gin.Context) {
	var q transport.ListAgentsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	filter := agentRepo.ListFilter{
		BaseFilter: query.BaseFilter{
			Search: q.Search,
			Order:  query.SortOrder(q.Order),
			Limit:  q.Limit,
		},
		Cursor: q.Cursor,
	}

	agents, nextCursor, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.log.WithError(err).Error("failed to list agents")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	items := make([]*transport.AgentResponse, 0, len(agents))
	for _, a := range agents {
		items = append(items, transport.ToAgentResponse(a))
	}

	response.Write(c, http.StatusOK, "agents retrieved", transport.ListAgentsResponse{
		Items: items,
		Metadata: transport.ListAgentsMetadata{
			NextCursor: nextCursor,
			HasMore:    nextCursor != "",
			Limit:      filter.Limit,
		},
	})
}

func (h *AgentHandler) Create(c *gin.Context) {
	var req transport.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	claims := middleware.UserClaimsFromContext(c)

	a, err := h.svc.Create(c.Request.Context(), agentService.CreateAgentInput{
		Name:        req.Name,
		Hostname:    req.Hostname,
		IPAddress:   req.IPAddress,
		Version:     req.Version,
		Status:      req.Status,
		AgentSecret: req.AgentSecret,
	}, claims, c.ClientIP())
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

	claims := middleware.UserClaimsFromContext(c)

	a, err := h.svc.Update(c.Request.Context(), id, agentService.UpdateAgentInput{
		Name:      req.Name,
		Hostname:  req.Hostname,
		IPAddress: req.IPAddress,
		Version:   req.Version,
		Status:    req.Status,
	}, claims, c.ClientIP())
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

	claims := middleware.UserClaimsFromContext(c)

	if err := h.svc.Delete(c.Request.Context(), id, claims, c.ClientIP()); err != nil {
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
