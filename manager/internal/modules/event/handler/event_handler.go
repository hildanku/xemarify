package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hildanku/xemarify/internal/infrastructure/metrics"
	"github.com/hildanku/xemarify/internal/infrastructure/middleware"
	eventRepo "github.com/hildanku/xemarify/internal/modules/event/repository"
	"github.com/hildanku/xemarify/internal/modules/event/service"
	"github.com/hildanku/xemarify/internal/modules/event/transport"
	"github.com/hildanku/xemarify/pkg/query"
	"github.com/hildanku/xemarify/pkg/response"
	"github.com/sirupsen/logrus"
)

const maxBodyBytes = 1 << 20 // 1 MB

// EventHandler handles HTTP requests for the event ingestion endpoint.
type EventHandler struct {
	svc     *service.EventService
	metrics *metrics.Metrics
	log     *logrus.Logger
}

// NewEventHandler creates an EventHandler with its dependencies.
func NewEventHandler(svc *service.EventService, m *metrics.Metrics, log *logrus.Logger) *EventHandler {
	return &EventHandler{svc: svc, metrics: m, log: log}
}

// Register wires the handler routes onto the given router group.
// Expected to be called with a group that already has auth + rate-limit middleware applied.
func (h *EventHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/events", h.Ingest)
}

// RegisterManager wires read-only event routes onto a manager-auth group.
func (h *EventHandler) RegisterManager(rg *gin.RouterGroup) {
	rg.GET("", h.List)
}

// Ingest handles POST /api/v1/events.
// Validates, normalises, and persists an event. Returns 202 on success.
func (h *EventHandler) Ingest(c *gin.Context) {
	start := time.Now()

	// Enforce maximum body size to prevent abuse.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	agent := middleware.AgentFromContext(c)
	if agent == nil {
		// Should never reach here; auth middleware guards this route.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req transport.EventBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.metrics.EventsFailed.WithLabelValues("validation_error").Inc()
		h.log.WithFields(logrus.Fields{
			"agent_id":    agent.ID,
			"remote_addr": c.ClientIP(),
		}).WithError(err).Warn("invalid event payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.metrics.EventsReceived.WithLabelValues(agent.ID.String()).Add(float64(len(req.Events)))

	accepted, err := h.svc.IngestBatch(c.Request.Context(), agent.ID, &req)
	if err != nil {
		h.metrics.EventsFailed.WithLabelValues("ingest_error").Inc()
		if errors.Is(err, service.ErrAgentIDMismatch) {
			c.JSON(http.StatusForbidden, gin.H{"error": "agent identity mismatch"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest event"})
		return
	}

	elapsed := time.Since(start).Seconds()
	h.metrics.IngestionLatency.Observe(elapsed)

	c.JSON(http.StatusAccepted, gin.H{
		"accepted": accepted,
	})
}

// List handles GET /api/v1/events (manager-only).
//
// Query params:
//
//	search    - case-insensitive partial match on hostname, severity, category (indexed columns only)
//	sort_by   - field to sort by (received_at|event_time|hostname|severity|category|created_at); default: received_at
//	order     - sort direction (asc|desc); default: desc
//	limit     - max rows (1-100); default: 10
//	offset    - rows to skip; default: 0
//	date_from - ISO-8601 lower bound on received_at; default: NOW()-30d (for partition pruning)
//	date_to   - ISO-8601 upper bound on received_at; default: NOW()
//	agent_id  - filter by agent UUID (optional)
//	severity  - exact severity filter (optional)
//	category  - exact category filter (optional)
func (h *EventHandler) List(c *gin.Context) {
	var q transport.ListEventsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	var agentID *string
	if q.AgentID != "" {
		agentID = &q.AgentID
	}

	filter := eventRepo.ListFilter{
		BaseFilter: query.BaseFilter{
			Search: q.Search,
			SortBy: q.SortBy,
			Order:  query.SortOrder(q.Order),
			Limit:  q.Limit,
			Offset: q.Offset,
		},
		DateFrom: q.DateFrom,
		DateTo:   q.DateTo,
		AgentID:  agentID,
		Severity: q.Severity,
		Category: q.Category,
	}

	events, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.log.WithError(err).Error("failed to list events")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	items := make([]*transport.EventResponse, 0, len(events))
	for _, e := range events {
		items = append(items, transport.ToEventResponse(e))
	}

	totalPages := 0
	if filter.Limit > 0 {
		totalPages = (total + filter.Limit - 1) / filter.Limit
	}

	response.Write(c, http.StatusOK, "events retrieved", transport.ListEventsResponse{
		Items: items,
		Metadata: transport.ListEventsMetadata{
			Total:      total,
			TotalPages: totalPages,
			Limit:      filter.Limit,
			Offset:     filter.Offset,
		},
	})
}
