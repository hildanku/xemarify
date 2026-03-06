package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	auditRepo "github.com/hildanku/xemarify/internal/modules/audit/repository"
	auditService "github.com/hildanku/xemarify/internal/modules/audit/service"
	"github.com/hildanku/xemarify/internal/modules/audit/transport"
	"github.com/hildanku/xemarify/pkg/response"
	"github.com/sirupsen/logrus"
)

// AuditLogHandler handles HTTP requests for the audit log endpoints.
type AuditLogHandler struct {
	svc *auditService.AuditLogService
	log *logrus.Logger
}

// NewAuditLogHandler constructs an AuditLogHandler.
func NewAuditLogHandler(svc *auditService.AuditLogService, log *logrus.Logger) *AuditLogHandler {
	return &AuditLogHandler{svc: svc, log: log}
}

// Register wires the audit log routes onto the given router group.
// The group must already have JWT + RBAC(MANAGER|ANALYST) middleware applied.
func (h *AuditLogHandler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
}

// List handles GET /api/v1/audit-logs.
// Supports optional query params: action, date_from, date_to, page, page_size.
func (h *AuditLogHandler) List(c *gin.Context) {
	var q transport.ListAuditLogsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	filter := auditRepo.ListFilter{
		Action:   q.Action,
		DateFrom: q.DateFrom,
		DateTo:   q.DateTo,
		Page:     q.Page,
		PageSize: q.PageSize,
	}

	entries, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.log.WithError(err).Error("failed to list audit logs")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	items := make([]*transport.AuditLogResponse, 0, len(entries))
	for _, e := range entries {
		items = append(items, transport.ToAuditLogResponse(e))
	}

	response.Write(c, http.StatusOK, "audit logs retrieved", transport.AuditLogListResponse{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	})
}
