package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/infrastructure/middleware"
	inventoryService "github.com/hildanku/xemarify/internal/modules/inventory/service"
	"github.com/hildanku/xemarify/internal/modules/inventory/transport"
	"github.com/hildanku/xemarify/pkg/response"
	"github.com/sirupsen/logrus"
)

// InventoryHandler handles HTTP requests for agent inventory endpoints.
type InventoryHandler struct {
	svc *inventoryService.InventoryService
	log *logrus.Logger
}

// NewInventoryHandler constructs an InventoryHandler.
func NewInventoryHandler(svc *inventoryService.InventoryService, log *logrus.Logger) *InventoryHandler {
	return &InventoryHandler{svc: svc, log: log}
}

// RegisterAgentSession wires agent-authenticated inventory routes.
// Must be called on a group that already has AgentAuth middleware applied.
func (h *InventoryHandler) RegisterAgentSession(rg *gin.RouterGroup) {
	rg.POST("/inventory", h.Upsert)
}

// Register wires manager-authenticated inventory routes.
// Must be called on a group that already has JWT + RBAC middleware applied.
func (h *InventoryHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/:id/inventory", h.GetByAgentID)
}

// Upsert handles POST /api/v1/agents/inventory.
// Called by the agent to push a fresh system snapshot.
func (h *InventoryHandler) Upsert(c *gin.Context) {
	authenticatedAgent := middleware.AgentFromContext(c)
	if authenticatedAgent == nil {
		response.Write(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req transport.InventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate that the agent_id in the body matches the authenticated agent.
	requestedAgentID, err := uuid.Parse(req.AgentID)
	if err != nil || requestedAgentID != authenticatedAgent.ID {
		response.Write(c, http.StatusForbidden, "agent identity mismatch", nil)
		return
	}

	collectedAt := req.CollectedAt
	if collectedAt.IsZero() {
		collectedAt = time.Now().UTC()
	}

	if err := h.svc.Upsert(c.Request.Context(), inventoryService.UpsertInput{
		AgentID:         authenticatedAgent.ID,
		OS:              req.OS,
		Arch:            req.Arch,
		KernelVersion:   req.KernelVersion,
		CPUModel:        req.CPUModel,
		CPUCores:        req.CPUCores,
		MemoryTotalMB:   req.MemoryTotalMB,
		UptimeSeconds:   req.UptimeSeconds,
		IPAddresses:     req.IPAddresses,
		NginxInstalled:  req.NginxInstalled,
		ApacheInstalled: req.ApacheInstalled,
		CollectedAt:     collectedAt,
	}); err != nil {
		h.log.WithError(err).Error("failed to upsert inventory")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "inventory accepted", nil)
}

// GetByAgentID handles GET /api/v1/agents/:id/inventory.
// Returns the latest inventory snapshot for the given agent.
func (h *InventoryHandler) GetByAgentID(c *gin.Context) {
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid agent id", nil)
		return
	}

	inv, err := h.svc.GetByAgentID(c.Request.Context(), agentID)
	if err != nil {
		if errors.Is(err, inventoryService.ErrInventoryNotFound) {
			response.Write(c, http.StatusNotFound, "inventory not found", nil)
			return
		}
		h.log.WithError(err).Error("failed to get inventory")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "inventory retrieved", transport.ToInventoryResponse(inv))
}
