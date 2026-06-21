package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/engine"
	"github.com/hildanku/xemarify/internal/infrastructure/metrics"
	eventDomain "github.com/hildanku/xemarify/internal/modules/event/domain"
	eventRepo "github.com/hildanku/xemarify/internal/modules/event/repository"
	"github.com/hildanku/xemarify/internal/modules/event/transport"
	ruleDomain "github.com/hildanku/xemarify/internal/modules/rule/domain"
	ruleRepo "github.com/hildanku/xemarify/internal/modules/rule/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	integrationDBHost     = "localhost"
	integrationDBPort     = 5445
	integrationDBUser     = "xemarify_manager"
	integrationDBPassword = "xemarify_manager"
	integrationDBName     = "xemarify_manager"
	integrationDBSSLMode  = "disable"
)

func newIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		integrationDBUser,
		integrationDBPassword,
		integrationDBHost,
		integrationDBPort,
		integrationDBName,
		integrationDBSSLMode,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping integration test - could not create pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping integration test - database unreachable: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

func TestEventIngestTriggersAlertIntegration(t *testing.T) {
	pool := newIntegrationPool(t)
	ctx := context.Background()
	logger := logrus.New()

	ruleRepository := ruleRepo.NewPgRuleRepository(pool)
	createdBy := uuid.New()
	ruleID := uuid.New()
	ruleName := "integration-test-web-401-" + uuid.New().String()[:8]

	rule := &ruleDomain.Rule{
		ID:          ruleID,
		Name:        ruleName,
		Description: "integration test rule for web_401",
		Level:       "HIGH",
		Enabled:     true,
		Tags:        []string{"integration-test"},
		Version:     1,
		CreatedBy:   &createdBy,
		Condition: ruleDomain.RuleCondition{
			Type:      "threshold",
			EventType: "web_401",
			Threshold: 1,
			WindowSec: 60,
		},
	}

	if err := ruleRepository.Create(ctx, rule); err != nil {
		t.Fatalf("failed to create test rule: %v", err)
	}

	messageMarker := "integration-test-web-401-message-" + uuid.New().String()
	cleanup := func() {
		_, _ = pool.Exec(ctx, "DELETE FROM alerts WHERE rule_id = $1", ruleID)
		_, _ = pool.Exec(ctx, "DELETE FROM rule_evaluations WHERE rule_id = $1", ruleID)
		_, _ = pool.Exec(ctx, "DELETE FROM correlation_state WHERE rule_id = $1", ruleID)
		_, _ = pool.Exec(ctx, "DELETE FROM detection_alert_dedup WHERE dedup_key LIKE $1", ruleID.String()+"%")
		_, _ = pool.Exec(ctx, "DELETE FROM rules WHERE id = $1", ruleID)
		_, _ = pool.Exec(ctx, "DELETE FROM events WHERE message = $1", messageMarker)
	}
	t.Cleanup(cleanup)

	ruleEngine, err := engine.NewRuleEngine(ctx, pool, logger)
	if err != nil {
		t.Fatalf("failed to initialize rule engine: %v", err)
	}
	metrics := &metrics.Metrics{
		DBInsertLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "xemarify_test_db_insert_duration_seconds",
			Help: "DB insert latency histogram for tests.",
		}),
		ChannelDepth: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "xemarify_test_event_channel_depth",
			Help: "Channel depth gauge for tests.",
		}),
	}

	eventRepository := eventRepo.NewPgEventRepository(pool)
	eventService := NewEventService(eventRepository, ruleEngine, nil, metrics, logger, 8, 4096)

	agentID := uuid.New()
	event := transport.IngestEvent{
		EventTime: time.Now().UTC(),
		Hostname:  "integration-test-host",
		SourceIP:  "10.10.0.10",
		InputType: "syslog",
		Facility:  "auth",
		Severity:  "HIGH",
		Category:  "system",
		Message:   "GET /admin 401 Unauthorized " + messageMarker,
		Raw:       "GET /admin 401 Unauthorized",
		Normalized: map[string]interface{}{
			"source_ip": "10.10.0.10",
		},
	}

	result, err := eventService.IngestBatch(ctx, agentID, &transport.EventBatchRequest{
		AgentID: agentID.String(),
		Events:  []transport.IngestEvent{event},
	})
	if err != nil {
		t.Fatalf("ingest batch failed: %v", err)
	}
	if result.Accepted != 1 {
		t.Fatalf("unexpected accepted count: %d", result.Accepted)
	}

	var storedType string
	if err := pool.QueryRow(ctx, "SELECT normalized->>'event_type' FROM events WHERE message = $1 ORDER BY received_at DESC LIMIT 1", event.Message).Scan(&storedType); err != nil {
		t.Fatalf("failed to query stored event: %v", err)
	}
	if storedType != "web_401" {
		t.Fatalf("stored event_type = %q, want %q", storedType, "web_401")
	}

	var alertCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM alerts WHERE rule_id = $1", ruleID).Scan(&alertCount); err != nil {
		t.Fatalf("failed to query alerts: %v", err)
	}
	if alertCount < 1 {
		t.Fatalf("expected alert for rule %s, got %d", ruleID, alertCount)
	}

	ruleEngine.Stop()
}

func TestEventDerivationIntegration_NoOverride(t *testing.T) {
	pool := newIntegrationPool(t)
	ctx := context.Background()
	logger := logrus.New()

	eventRepository := eventRepo.NewPgEventRepository(pool)
	metrics := &metrics.Metrics{
		DBInsertLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "xemarify_test_db_insert_duration_seconds_override",
			Help: "DB insert latency histogram for tests.",
		}),
		ChannelDepth: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "xemarify_test_event_channel_depth_override",
			Help: "Channel depth gauge for tests.",
		}),
	}

	eventService := NewEventService(eventRepository, nil, nil, metrics, logger, 8, 4096)
	agentID := uuid.New()
	messageMarker := "integration-test-explicit-event-type-" + uuid.New().String()

	event := transport.IngestEvent{
		EventTime: time.Now().UTC(),
		Hostname:  "integration-test-host",
		SourceIP:  "10.10.0.11",
		InputType: "syslog",
		Facility:  "auth",
		Severity:  "HIGH",
		Category:  "system",
		Message:   "GET /admin 401 Unauthorized " + messageMarker,
		Raw:       "GET /admin 401 Unauthorized",
		Normalized: map[string]interface{}{
			"event_type": "custom_type",
		},
	}

	result, err := eventService.IngestBatch(ctx, agentID, &transport.EventBatchRequest{
		AgentID: agentID.String(),
		Events:  []transport.IngestEvent{event},
	})
	if err != nil {
		t.Fatalf("ingest batch failed: %v", err)
	}
	if result.Accepted != 1 {
		t.Fatalf("unexpected accepted count: %d", result.Accepted)
	}

	var storedType string
	if err := pool.QueryRow(ctx, "SELECT normalized->>'event_type' FROM events WHERE message = $1 ORDER BY received_at DESC LIMIT 1", event.Message).Scan(&storedType); err != nil {
		t.Fatalf("failed to query stored event: %v", err)
	}
	if storedType != "custom_type" {
		t.Fatalf("stored event_type = %q, want %q", storedType, "custom_type")
	}

	_, _ = pool.Exec(ctx, "DELETE FROM events WHERE message = $1", messageMarker)
}

func TestEventDerivationIntegration_FromNormalizedStatus(t *testing.T) {
	pool := newIntegrationPool(t)
	ctx := context.Background()
	logger := logrus.New()

	eventRepository := eventRepo.NewPgEventRepository(pool)
	metrics := &metrics.Metrics{
		DBInsertLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "xemarify_test_db_insert_duration_seconds_status",
			Help: "DB insert latency histogram for tests.",
		}),
		ChannelDepth: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "xemarify_test_event_channel_depth_status",
			Help: "Channel depth gauge for tests.",
		}),
	}

	eventService := NewEventService(eventRepository, nil, nil, metrics, logger, 8, 4096)
	agentID := uuid.New()
	messageMarker := "integration-test-normalized-status-" + uuid.New().String()

	event := transport.IngestEvent{
		EventTime: time.Now().UTC(),
		Hostname:  "integration-test-host",
		SourceIP:  "10.10.0.12",
		InputType: "syslog",
		Facility:  "auth",
		Severity:  "HIGH",
		Category:  "system",
		Message:   "test event " + messageMarker,
		Raw:       "test raw",
		Normalized: map[string]interface{}{
			"status": 401,
		},
	}

	result, err := eventService.IngestBatch(ctx, agentID, &transport.EventBatchRequest{
		AgentID: agentID.String(),
		Events:  []transport.IngestEvent{event},
	})
	if err != nil {
		t.Fatalf("ingest batch failed: %v", err)
	}
	if result.Accepted != 1 {
		t.Fatalf("unexpected accepted count: %d", result.Accepted)
	}

	var storedType string
	if err := pool.QueryRow(ctx, "SELECT normalized->>'event_type' FROM events WHERE message = $1 ORDER BY received_at DESC LIMIT 1", event.Message).Scan(&storedType); err != nil {
		t.Fatalf("failed to query stored event: %v", err)
	}
	if storedType != "web_401" {
		t.Fatalf("stored event_type = %q, want %q", storedType, "web_401")
	}

	_, _ = pool.Exec(ctx, "DELETE FROM events WHERE message = $1", messageMarker)
}

func newTestEvent(eventType string) *eventDomain.Event {
	return &eventDomain.Event{
		EventTime:  time.Now().UTC(),
		ReceivedAt: time.Now().UTC(),
		AgentID:    uuid.New(),
		Hostname:   "integration-test-host",
		SourceIP:   "10.10.0.99",
		InputType:  "syslog",
		Facility:   "auth",
		Severity:   "HIGH",
		Category:   "system",
		Message:    "integration-test",
		Raw:        "integration-test",
		Normalized: map[string]interface{}{"event_type": eventType},
	}
}
