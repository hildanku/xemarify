package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/engine"
	"github.com/hildanku/xemarify/internal/infrastructure/metrics"
	"github.com/hildanku/xemarify/internal/infrastructure/sse"
	"github.com/hildanku/xemarify/internal/modules/event/domain"
	eventRepo "github.com/hildanku/xemarify/internal/modules/event/repository"
	"github.com/hildanku/xemarify/internal/modules/event/transport"
	"github.com/sirupsen/logrus"
)

const (
	defaultEventWorkerCount = 8
	defaultEventChanBuffer  = 4096
)

var ErrAgentIDMismatch = errors.New("agent id mismatch")

var ErrEventNotFound = errors.New("event not found")

type EventService struct {
	eventRepo   eventRepo.EventRepository
	engine      engine.Engine
	hub         *sse.Hub
	metrics     *metrics.Metrics
	log         *logrus.Logger

	eventCh     chan *domain.Event
	workerWG    sync.WaitGroup
	workerCount int
	chanBuffer  int
}

func NewEventService(
	eventRepo eventRepo.EventRepository,
	detectionEngine engine.Engine,
	hub *sse.Hub,
	m *metrics.Metrics,
	log *logrus.Logger,
	workerCount int,
	chanBuffer int,
) *EventService {
	if workerCount <= 0 {
		workerCount = defaultEventWorkerCount
	}
	if chanBuffer <= 0 {
		chanBuffer = defaultEventChanBuffer
	}

	return &EventService{
		eventRepo:   eventRepo,
		engine:      detectionEngine,
		hub:         hub,
		metrics:     m,
		log:         log,
		eventCh:     make(chan *domain.Event, chanBuffer),
		workerCount: workerCount,
		chanBuffer:  chanBuffer,
	}
}

func (s *EventService) Start() {
	for i := 0; i < s.workerCount; i++ {
		s.workerWG.Add(1)
		go s.processLoop()
	}
	s.log.WithField("worker_count", s.workerCount).Info("event processing workers started")
}

func (s *EventService) Stop() {
	close(s.eventCh)
	s.workerWG.Wait()
	s.log.Info("event processing workers stopped, channel drained")
}

func (s *EventService) processLoop() {
	defer s.workerWG.Done()
	for event := range s.eventCh {
		ctx := context.Background()
		if s.engine != nil {
			if err := s.engine.ProcessEvent(ctx, event); err != nil {
				s.log.WithFields(logrus.Fields{
					"event_id": event.ID,
					"agent_id": event.AgentID,
				}).WithError(err).Warn("rule engine processing failed")
			}
		}
		if s.hub != nil {
			s.hub.Broadcast("new_event", transport.ToEventResponse(event))
		}
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

	// normalize event
	events := make([]*domain.Event, 0, len(req.Events))
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
		events = append(events, event)
	}

	dbStart := time.Now()
	if err := s.eventRepo.BatchInsert(ctx, events); err != nil {
		s.log.WithField("agent_id", authenticatedAgentID).WithError(err).Error("failed to batch insert events")
		return 0, fmt.Errorf("db batch insert failed: %w", err)
	}
	s.metrics.DBInsertLatency.Observe(time.Since(dbStart).Seconds())

	for _, event := range events {
		select {
		case s.eventCh <- event:
		default:
			s.metrics.EventsFailed.WithLabelValues("channel_full").Inc()
			s.log.WithField("event_id", event.ID).Warn("event processing channel full, event will not be processed by rule engine")
		}
	}

	return len(events), nil
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

	ensureEventType(e)
}

func ensureEventType(e *domain.Event) {
	if e == nil || e.Normalized == nil {
		return
	}
	if _, ok := e.Normalized["event_type"]; ok {
		return
	}
	eventType := deriveEventType(e)
	if eventType == "" {
		return
	}
	e.Normalized["event_type"] = eventType
}

func deriveEventType(e *domain.Event) string {
	if e == nil {
		return ""
	}

	if statusValue, ok := normalizedString(e.Normalized, "status"); ok {
		if eventType := mapHTTPStatus(statusValue); eventType != "" {
			return eventType
		}
	}
	if statusValue, ok := normalizedString(e.Normalized, "http_status"); ok {
		if eventType := mapHTTPStatus(statusValue); eventType != "" {
			return eventType
		}
	}

	message := strings.ToLower(strings.TrimSpace(e.Message))
	raw := strings.ToLower(strings.TrimSpace(e.Raw))
	combined := strings.TrimSpace(strings.Join([]string{message, raw}, " "))

	if strings.Contains(combined, "sudo") && (strings.Contains(combined, "authentication failure") || strings.Contains(combined, "incorrect password")) {
		return "sudo_failed"
	}
	if strings.Contains(combined, "sudo") && (strings.Contains(combined, "session opened") || strings.Contains(combined, "command=") || strings.Contains(combined, "sudo:")) {
		return "sudo_used"
	}
	if strings.Contains(combined, "invalid user") {
		return "ssh_invalid_user"
	}
	if strings.Contains(combined, "failed password") || strings.Contains(combined, "authentication failure") || strings.Contains(combined, "login failed") {
		return "login_failed"
	}
	if strings.Contains(combined, "accepted password") || strings.Contains(combined, "login success") || strings.Contains(combined, "login succeeded") {
		return "login_success"
	}
	if strings.Contains(combined, "privilege escalation") || strings.Contains(combined, "elevated privileges") {
		return "privilege_escalation"
	}
	if strings.Contains(combined, "port scan") || strings.Contains(combined, "nmap") {
		return "port_scan_detected"
	}
	if strings.Contains(combined, "suspicious process") || strings.Contains(combined, "process exec") || strings.Contains(combined, "malware") {
		return "process_exec_suspicious"
	}
	if strings.Contains(combined, "service installed") || strings.Contains(combined, "apt install") || strings.Contains(combined, "yum install") {
		return "service_installed"
	}
	if strings.Contains(combined, "service started") || strings.Contains(combined, "systemd started") {
		return "service_started"
	}
	if strings.Contains(combined, "useradd") || strings.Contains(combined, "user created") {
		return "user_created"
	}
	if strings.Contains(combined, "file integrity") || strings.Contains(combined, "integrity violation") {
		return "file_integrity_changed"
	}

	if eventType := mapHTTPStatus(combined); eventType != "" {
		return eventType
	}
	if strings.Contains(combined, "web login failed") || strings.Contains(combined, "invalid credentials") {
		return "web_login_failed"
	}
	if strings.Contains(combined, "web login success") || strings.Contains(combined, "login successful") {
		return "web_login_success"
	}

	if actionValue, ok := normalizedString(e.Normalized, "action"); ok {
		if actionValue == "login" {
			return "login_success"
		}
	}

	return ""
}

func normalizedString(values map[string]interface{}, key string) (string, bool) {
	if values == nil {
		return "", false
	}
	raw, ok := values[key]
	if !ok || raw == nil {
		return "", false
	}
	switch v := raw.(type) {
	case string:
		value := strings.TrimSpace(strings.ToLower(v))
		if value == "" {
			return "", false
		}
		return value, true
	case int:
		return strings.ToLower(strings.TrimSpace(fmt.Sprintf("%d", v))), true
	case int32:
		return strings.ToLower(strings.TrimSpace(fmt.Sprintf("%d", v))), true
	case int64:
		return strings.ToLower(strings.TrimSpace(fmt.Sprintf("%d", v))), true
	case float64:
		return strings.ToLower(strings.TrimSpace(fmt.Sprintf("%.0f", v))), true
	default:
		return "", false
	}
}

func mapHTTPStatus(value string) string {
	if value == "" {
		return ""
	}
	if strings.Contains(value, "401") {
		return "web_401"
	}
	if strings.Contains(value, "403") {
		return "web_403"
	}
	if strings.Contains(value, "500") {
		return "web_500"
	}
	return ""
}

// List returns a filtered, sorted, paginated slice of events and the total
// match count within the requested date window.
func (s *EventService) List(ctx context.Context, filter eventRepo.ListFilter) ([]*domain.Event, int, error) {
	return s.eventRepo.List(ctx, filter)
}

// GetByID returns one event by ID.
func (s *EventService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrEventNotFound
	}
	return event, nil
}
