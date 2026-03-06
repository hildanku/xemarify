package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hildanku/xemarify/internal/infrastructure/metrics"
	"github.com/hildanku/xemarify/internal/infrastructure/middleware"
	"github.com/hildanku/xemarify/internal/modules/event/service"
	"github.com/hildanku/xemarify/internal/modules/event/transport"
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

	var req transport.IngestEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.metrics.EventsFailed.WithLabelValues("validation_error").Inc()
		h.log.WithFields(logrus.Fields{
			"agent_id":    agent.ID,
			"remote_addr": c.ClientIP(),
		}).WithError(err).Warn("invalid event payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.metrics.EventsReceived.WithLabelValues(agent.ID.String()).Inc()

	event, err := h.svc.Ingest(c.Request.Context(), agent.ID, &req)
	if err != nil {
		h.metrics.EventsFailed.WithLabelValues("ingest_error").Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest event"})
		return
	}

	elapsed := time.Since(start).Seconds()
	h.metrics.IngestionLatency.Observe(elapsed)

	c.JSON(http.StatusAccepted, gin.H{
		"event_id":    event.ID,
		"received_at": event.ReceivedAt,
	})
}
