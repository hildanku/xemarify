package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/engine"
	"github.com/hildanku/xemarify/internal/infrastructure/metrics"
	"github.com/hildanku/xemarify/internal/modules/event/domain"
	eventRepo "github.com/hildanku/xemarify/internal/modules/event/repository"
	"github.com/hildanku/xemarify/internal/modules/event/transport"
	"github.com/sirupsen/logrus"
)

var ErrAgentIDMismatch = errors.New("agent id mismatch")

// EventService orchestrates event validation, normalization, and persistence.
// It owns a single public method - Ingest - which is the intake point for the
// ingestion pipeline.  All steps are synchronous (Phase 1 design decision).
type EventService struct {
	eventRepo eventRepo.EventRepository
	engine    engine.Engine
	metrics   *metrics.Metrics
	log       *logrus.Logger
}

// NewEventService constructs the service with its required dependencies.
func NewEventService(
	eventRepo eventRepo.EventRepository,
	detectionEngine engine.Engine,
	m *metrics.Metrics,
	log *logrus.Logger,
) *EventService {
	return &EventService{
		eventRepo: eventRepo,
		engine:    detectionEngine,
		metrics:   m,
		log:       log,
	}
}

// IngestBatch validates, normalizes, and persists an event batch.
func (s *EventService) IngestBatch(ctx context.Context, authenticatedAgentID uuid.UUID, req *transport.EventBatchRequest) (int, error) {
	batchAgentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		return 0, fmt.Errorf("invalid agent id: %w", err)
	}
	if batchAgentID != authenticatedAgentID {
		return 0, ErrAgentIDMismatch
	}

	accepted := 0
	for _, item := range req.Events {
		receivedAt := time.Now().UTC()
		eventTime := receivedAt
		if !item.EventTime.IsZero() {
			eventTime = item.EventTime.UTC()
		}

		event := &domain.Event{
			ID:         uuid.New(),
			EventTime:  eventTime,
			ReceivedAt: receivedAt,
			AgentID:    authenticatedAgentID,
			Hostname:   item.Hostname,
			SourceIP:   item.SourceIP,
			InputType:  item.InputType,
			Facility:   item.Facility,
			Severity:   item.Severity,
			Category:   item.Category,
			Message:    item.Message,
			Raw:        item.Raw,
			Normalized: item.Normalized,
		}
		if event.Normalized == nil {
			event.Normalized = make(map[string]interface{})
		}

		s.normalize(event)

		dbStart := time.Now()
		if err := s.eventRepo.Insert(ctx, event); err != nil {
			s.log.WithFields(logrus.Fields{
				"event_id": event.ID,
				"agent_id": authenticatedAgentID,
			}).WithError(err).Error("failed to insert event")
			return accepted, fmt.Errorf("db insert failed: %w", err)
		}
		s.metrics.DBInsertLatency.Observe(time.Since(dbStart).Seconds())

		if s.engine != nil {
			if err := s.engine.ProcessEvent(ctx, event); err != nil {
				s.log.WithFields(logrus.Fields{
					"event_id": event.ID,
					"agent_id": authenticatedAgentID,
				}).WithError(err).Warn("rule engine processing failed")
			}
		}

		accepted++
	}

	return accepted, nil
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
