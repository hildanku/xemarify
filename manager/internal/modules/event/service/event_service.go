package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/engine"
	"github.com/hildanku/xemarify/internal/infrastructure/metrics"
	agentRepo "github.com/hildanku/xemarify/internal/modules/agent/repository"
	"github.com/hildanku/xemarify/internal/modules/event/domain"
	eventRepo "github.com/hildanku/xemarify/internal/modules/event/repository"
	"github.com/hildanku/xemarify/internal/modules/event/transport"
	"github.com/sirupsen/logrus"
)

// EventService orchestrates event validation, normalization, and persistence.
// It owns a single public method - Ingest - which is the intake point for the
// ingestion pipeline.  All steps are synchronous (Phase 1 design decision).
type EventService struct {
	eventRepo eventRepo.EventRepository
	agentRepo agentRepo.AgentRepository
	engine    engine.Engine
	metrics   *metrics.Metrics
	log       *logrus.Logger
}

// NewEventService constructs the service with its required dependencies.
func NewEventService(
	eventRepo eventRepo.EventRepository,
	agentRepo agentRepo.AgentRepository,
	detectionEngine engine.Engine,
	m *metrics.Metrics,
	log *logrus.Logger,
) *EventService {
	return &EventService{
		eventRepo: eventRepo,
		agentRepo: agentRepo,
		engine:    detectionEngine,
		metrics:   m,
		log:       log,
	}
}

// Ingest validates, normalises, and persists an event.
// It also updates the agent's last_seen_at after a successful insert.
// Returns an error on any failure. Callers should NOT ack the agent on error.
func (s *EventService) Ingest(ctx context.Context, agentID uuid.UUID, req *transport.IngestEventRequest) (*domain.Event, error) {
	eventID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid event id: %w", err)
	}

	receivedAt := time.Now().UTC()
	eventTime := receivedAt
	if req.EventTime != "" {
		parsed, err := time.Parse(time.RFC3339Nano, req.EventTime)
		if err != nil {
			s.log.WithFields(logrus.Fields{
				"event_id":   eventID,
				"agent_id":   agentID,
				"event_time": req.EventTime,
			}).Warn("invalid event_time format, falling back to received_at")
		} else {
			eventTime = parsed
		}
	}

	event := &domain.Event{
		ID:         eventID,
		EventTime:  eventTime,
		ReceivedAt: receivedAt,
		AgentID:    agentID,
		Hostname:   req.Hostname,
		SourceIP:   req.SourceIP,
		InputType:  req.InputType,
		Facility:   req.Facility,
		Severity:   req.Severity,
		Category:   req.Category,
		Message:    req.Message,
		Raw:        req.Raw,
		Normalized: req.Normalized,
	}
	if event.Normalized == nil {
		event.Normalized = make(map[string]interface{})
	}

	s.normalize(event)

	dbStart := time.Now()
	if err := s.eventRepo.Insert(ctx, event); err != nil {
		s.log.WithFields(logrus.Fields{
			"event_id": eventID,
			"agent_id": agentID,
		}).WithError(err).Error("failed to insert event")
		return nil, fmt.Errorf("db insert failed: %w", err)
	}
	s.metrics.DBInsertLatency.Observe(time.Since(dbStart).Seconds())

	if err := s.agentRepo.UpdateLastSeen(ctx, agentID); err != nil {
		s.log.WithFields(logrus.Fields{
			"agent_id": agentID,
		}).WithError(err).Warn("failed to update agent last_seen_at")
	}

	if s.engine != nil {
		if err := s.engine.ProcessEvent(ctx, event); err != nil {
			s.log.WithFields(logrus.Fields{
				"event_id": eventID,
				"agent_id": agentID,
			}).WithError(err).Warn("rule engine processing failed")
		}
	}

	s.log.WithFields(logrus.Fields{
		"event_id": eventID,
		"agent_id": agentID,
		"duration": time.Since(receivedAt).Milliseconds(),
	}).Info("event ingested successfully")

	return event, nil
}

// normalize enriches the event's Normalized map with fields from the top-level
// envelope that the rule engine will need later, ensuring consistency.
func (s *EventService) normalize(e *domain.Event) {
	if _, ok := e.Normalized["source_ip"]; !ok && e.SourceIP != "" {
		e.Normalized["source_ip"] = e.SourceIP
	}
	if _, ok := e.Normalized["hostname"]; !ok && e.Hostname != "" {
		e.Normalized["hostname"] = e.Hostname
	}
	if _, ok := e.Normalized["severity"]; !ok && e.Severity != "" {
		e.Normalized["severity"] = e.Severity
	}
	if _, ok := e.Normalized["category"]; !ok && e.Category != "" {
		e.Normalized["category"] = e.Category
	}
	if _, ok := e.Normalized["facility"]; !ok && e.Facility != "" {
		e.Normalized["facility"] = e.Facility
	}
}

// List returns a filtered, sorted, paginated slice of events and the total
// match count within the requested date window.
func (s *EventService) List(ctx context.Context, filter eventRepo.ListFilter) ([]*domain.Event, int, error) {
	return s.eventRepo.List(ctx, filter)
}
