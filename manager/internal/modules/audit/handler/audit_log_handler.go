package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	auditRepo "github.com/hildanku/xemarify/internal/modules/audit/repository"
	auditService "github.com/hildanku/xemarify/internal/modules/audit/service"
	"github.com/hildanku/xemarify/internal/modules/audit/transport"
	"github.com/hildanku/xemarify/pkg/query"
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
//
// Query params:
//
//	action    - exact action filter (optional)
//	date_from - ISO-8601 lower bound on created_at (optional)
//	date_to   - ISO-8601 upper bound on created_at (optional)
//	search    - case-insensitive partial match on action and user_identifier
//	sort_by   - field to sort by (action|user_identifier|created_at); default: created_at
//	order     - sort direction (asc|desc); default: desc
//	limit     - max rows (1-100); default: 10
//	offset    - rows to skip; default: 0
func (h *AuditLogHandler) List(c *gin.Context) {
	var q transport.ListAuditLogsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	filter := auditRepo.ListFilter{
		BaseFilter: query.BaseFilter{
			Search: q.Search,
			SortBy: q.SortBy,
			Order:  query.SortOrder(q.Order),
			Limit:  q.Limit,
			Offset: q.Offset,
		},
		Action:   q.Action,
		DateFrom: q.DateFrom,
		DateTo:   q.DateTo,
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

	totalPages := 0
	if filter.Limit > 0 {
		totalPages = (total + filter.Limit - 1) / filter.Limit
	}

	response.Write(c, http.StatusOK, "audit logs retrieved", transport.AuditLogListResponse{
		Items: items,
		Metadata: transport.AuditLogListMetadata{
			Total:      total,
			TotalPages: totalPages,
			Limit:      filter.Limit,
			Offset:     filter.Offset,
		},
	})
}
