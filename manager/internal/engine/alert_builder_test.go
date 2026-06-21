package engine

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	eventDomain "github.com/hildanku/xemarify/internal/modules/event/domain"
	"github.com/sirupsen/logrus"
)

var testEngineMetrics = NewEngineMetrics()

func newTestAlertBuilder() *PGAlertBuilder {
	b := &PGAlertBuilder{
		db:             nil,
		log:            logrus.New(),
		metrics:        testEngineMetrics,
		alertBuf:       make([]*Alert, 0, defaultAlertFlushBatchSize),
		flushInterval:  defaultAlertFlushInterval,
		flushBatchSize: 100000,
		stopCh:         make(chan struct{}),
	}
	return b
}

func makeTestAlert() *Alert {
	return &Alert{
		ID:             uuid.New(),
		RuleID:         uuid.New(),
		Severity:       "high",
		CorrelationKey: "host:web-01",
		TriggeredAt:    time.Now().UTC(),
		EventID:        uuid.New(),
		ReceivedAt:     time.Now().UTC(),
	}
}

func TestBuild_CreatesCorrectAlert(t *testing.T) {
	b := newTestAlertBuilder()
	ruleID := uuid.New()
	rule := CompiledRule{
		ID:       ruleID,
		Severity: "critical",
		Type:     "threshold",
	}
	state := State{
		LastSeen: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	}
	event := &eventDomain.Event{
		ID:         uuid.New(),
		ReceivedAt: time.Date(2026, 6, 20, 12, 1, 0, 0, time.UTC),
	}

	alert := b.Build(rule, "host:db-01", state, event)

	if alert.RuleID != ruleID {
		t.Fatalf("RuleID mismatch: got %v, want %v", alert.RuleID, ruleID)
	}
	if alert.Severity != "critical" {
		t.Fatalf("Severity mismatch: got %v, want critical", alert.Severity)
	}
	if alert.CorrelationKey != "host:db-01" {
		t.Fatalf("CorrelationKey mismatch: got %v, want host:db-01", alert.CorrelationKey)
	}
	if alert.TriggeredAt != state.LastSeen {
		t.Fatalf("TriggeredAt mismatch: got %v, want %v", alert.TriggeredAt, state.LastSeen)
	}
	if alert.EventID != event.ID {
		t.Fatalf("EventID mismatch: got %v, want %v", alert.EventID, event.ID)
	}
	if alert.ReceivedAt != event.ReceivedAt {
		t.Fatalf("ReceivedAt mismatch: got %v, want %v", alert.ReceivedAt, event.ReceivedAt)
	}
}

func TestBuild_ZeroTimeDefaults(t *testing.T) {
	b := newTestAlertBuilder()
	rule := CompiledRule{ID: uuid.New(), Severity: "low"}
	state := State{}
	event := &eventDomain.Event{ID: uuid.New()}

	before := time.Now().UTC()
	alert := b.Build(rule, "ck", state, event)

	if alert.TriggeredAt.Before(before) {
		t.Fatalf("TriggeredAt should default to now, got %v (before %v)", alert.TriggeredAt, before)
	}
	if alert.ReceivedAt.Before(before) {
		t.Fatalf("ReceivedAt should default to TriggeredAt, got %v", alert.ReceivedAt)
	}
}

func TestPersistBuffersSingleAlert(t *testing.T) {
	b := newTestAlertBuilder()
	alert := makeTestAlert()

	err := b.Persist(context.Background(), alert)
	if err != nil {
		t.Fatalf("Persist returned unexpected error: %v", err)
	}

	b.alertMu.Lock()
	bufLen := len(b.alertBuf)
	b.alertMu.Unlock()

	if bufLen != 1 {
		t.Fatalf("expected 1 alert in buffer, got %d", bufLen)
	}
}

func TestPersistAccumulatesMultipleAlerts(t *testing.T) {
	b := newTestAlertBuilder()
	const n = 10

	for i := 0; i < n; i++ {
		a := makeTestAlert()
		if err := b.Persist(context.Background(), a); err != nil {
			t.Fatalf("Persist %d returned error: %v", i, err)
		}
	}

	b.alertMu.Lock()
	bufLen := len(b.alertBuf)
	b.alertMu.Unlock()

	if bufLen != n {
		t.Fatalf("expected %d alerts in buffer, got %d", n, bufLen)
	}
}

func TestPersistConcurrentNoDataLoss(t *testing.T) {
	b := newTestAlertBuilder()
	const n = 100

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a := makeTestAlert()
			if err := b.Persist(context.Background(), a); err != nil {
				t.Errorf("Persist error: %v", err)
			}
		}()
	}
	wg.Wait()

	b.alertMu.Lock()
	bufLen := len(b.alertBuf)
	b.alertMu.Unlock()

	if bufLen != n {
		t.Fatalf("expected %d alerts in buffer after concurrent Persist, got %d", n, bufLen)
	}
}

func TestPersistReturnsImmediately(t *testing.T) {
	b := newTestAlertBuilder()
	alert := makeTestAlert()

	start := time.Now()
	err := b.Persist(context.Background(), alert)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Persist returned error: %v", err)
	}
	if elapsed > 10*time.Millisecond {
		t.Fatalf("Persist should return immediately (<10ms), took %v", elapsed)
	}
}
