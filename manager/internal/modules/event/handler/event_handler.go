package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/infrastructure/metrics"
	"github.com/hildanku/xemarify/internal/infrastructure/middleware"
	"github.com/hildanku/xemarify/internal/infrastructure/sse"
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
	hub     *sse.Hub
	metrics *metrics.Metrics
	log     *logrus.Logger
}

// NewEventHandler creates an EventHandler with its dependencies.
func NewEventHandler(svc *service.EventService, hub *sse.Hub, m *metrics.Metrics, log *logrus.Logger) *EventHandler {
	return &EventHandler{svc: svc, hub: hub, metrics: m, log: log}
}

// Register wires the handler routes onto the given router group.
// Expected to be called with a group that already has auth + rate-limit middleware applied.
func (h *EventHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/events", h.Ingest)
}

// RegisterManager wires read-only event routes onto a manager-auth group.
func (h *EventHandler) RegisterManager(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.GET("/:id", h.GetByID)
}

// RegisterStream wires the SSE stream endpoint onto a group with query-param auth.
func (h *EventHandler) RegisterStream(rg *gin.RouterGroup) {
	rg.GET("/stream", h.Stream)
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
//	order     - sort direction (asc|desc); default: desc
//	limit     - max rows (1-100); default: 10
//	cursor    - opaque keyset pagination token from previous response's next_cursor field
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
			Order:  query.SortOrder(q.Order),
			Limit:  q.Limit,
		},
		DateFrom: q.DateFrom,
		DateTo:   q.DateTo,
		AgentID:  agentID,
		Severity: q.Severity,
		Category: q.Category,
		Cursor:   q.Cursor,
	}

	events, nextCursor, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.log.WithError(err).Error("failed to list events")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	items := make([]*transport.EventResponse, 0, len(events))
	for _, e := range events {
		items = append(items, transport.ToEventResponse(e))
	}

	response.Write(c, http.StatusOK, "events retrieved", transport.ListEventsResponse{
		Items: items,
		Metadata: transport.ListEventsMetadata{
			NextCursor: nextCursor,
			HasMore:    nextCursor != "",
			Limit:      filter.Limit,
		},
	})
}

// GetByID handles GET /api/v1/events/:id (manager/analyst).
func (h *EventHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid event id", nil)
		return
	}

	event, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			response.Write(c, http.StatusNotFound, "event not found", nil)
			return
		}
		h.log.WithError(err).Error("failed to get event")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "event retrieved", transport.ToEventDetailResponse(event))
}

// Stream handles GET /api/v1/events/stream (SSE endpoint).
// Clients connect and receive real-time event notifications as they are ingested.
// A heartbeat comment is sent every 30 seconds to keep the connection alive.
func (h *EventHandler) Stream(c *gin.Context) {
	// Set SSE headers.
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Disable the server's WriteTimeout for this long-lived connection.
	rc := http.NewResponseController(c.Writer)
	_ = rc.SetWriteDeadline(time.Time{})

	// Write status and flush headers immediately so the client sees the connection open.
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	// Register this client with the hub.
	clientID := fmt.Sprintf("sse-%s", uuid.New().String()[:8])
	client := h.hub.Register(clientID)
	defer h.hub.Unregister(client)

	h.log.WithField("client_id", clientID).Debug("SSE client connected")

	// Heartbeat ticker to keep connection alive through proxies.
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			h.log.WithField("client_id", clientID).Debug("SSE client disconnected")
			return

		case msg, ok := <-client.Events:
			if !ok {
				// Hub closed the channel (shutdown).
				return
			}
			_, err := c.Writer.Write(msg)
			if err != nil {
				return
			}
			c.Writer.Flush()

		case <-ticker.C:
			// Send heartbeat comment to keep connection alive.
			_, err := c.Writer.Write([]byte(": heartbeat\n\n"))
			if err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}
