package engine

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	eventDomain "github.com/hildanku/xemarify/internal/modules/event/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

const (
	defaultAlertFlushInterval = 5 * time.Second
	defaultAlertFlushBatchSize = 50
)

type Alert struct {
	ID             uuid.UUID
	RuleID         uuid.UUID
	Severity       string
	CorrelationKey string
	TriggeredAt    time.Time
	EventID        uuid.UUID
	ReceivedAt     time.Time
}

type AlertWriter interface {
	Build(rule CompiledRule, correlationKey string, state State, event *eventDomain.Event) *Alert
	Persist(ctx context.Context, alert *Alert) error
}

type PGAlertBuilder struct {
	db             *pgxpool.Pool
	log            *logrus.Logger
	metrics        *EngineMetrics
	alertMu        sync.Mutex
	alertBuf       []*Alert
	flushInterval  time.Duration
	flushBatchSize int
	stopCh         chan struct{}
	stopWG         sync.WaitGroup
}

func NewPGAlertBuilder(db *pgxpool.Pool, log *logrus.Logger, metrics *EngineMetrics) *PGAlertBuilder {
	return &PGAlertBuilder{
		db:             db,
		log:            log,
		metrics:        metrics,
		alertBuf:       make([]*Alert, 0, defaultAlertFlushBatchSize),
		flushInterval:  defaultAlertFlushInterval,
		flushBatchSize: defaultAlertFlushBatchSize,
		stopCh:         make(chan struct{}),
	}
}

func (b *PGAlertBuilder) Build(rule CompiledRule, correlationKey string, state State, event *eventDomain.Event) *Alert {
	triggeredAt := state.LastSeen
	if triggeredAt.IsZero() {
		triggeredAt = time.Now().UTC()
	}

	receivedAt := event.ReceivedAt
	if receivedAt.IsZero() {
		receivedAt = triggeredAt
	}

	return &Alert{
		ID:             uuid.New(),
		RuleID:         rule.ID,
		Severity:       rule.Severity,
		CorrelationKey: correlationKey,
		TriggeredAt:    triggeredAt,
		EventID:        event.ID,
		ReceivedAt:     receivedAt,
	}
}

func (b *PGAlertBuilder) Persist(ctx context.Context, alert *Alert) error {
	b.alertMu.Lock()
	b.alertBuf = append(b.alertBuf, alert)
	bufLen := len(b.alertBuf)
	b.alertMu.Unlock()

	b.metrics.AlertBufferDepth.Inc()

	if bufLen >= b.flushBatchSize {
		b.Flush(ctx)
	}
	return nil
}

func (b *PGAlertBuilder) Start() {
	b.stopWG.Add(1)
	go b.periodicFlush()
	b.log.WithField("flush_interval", b.flushInterval).Info("alert batch writer started")
}

func (b *PGAlertBuilder) Stop() {
	close(b.stopCh)
	b.stopWG.Wait()
	b.Flush(context.Background())
	b.log.Info("alert batch writer stopped, final flush complete")
}

func (b *PGAlertBuilder) periodicFlush() {
	defer b.stopWG.Done()
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.Flush(context.Background())
		case <-b.stopCh:
			return
		}
	}
}

func (b *PGAlertBuilder) Flush(ctx context.Context) {
	b.alertMu.Lock()
	alerts := b.alertBuf
	b.alertBuf = make([]*Alert, 0, defaultAlertFlushBatchSize)
	b.alertMu.Unlock()

	if len(alerts) == 0 {
		return
	}

	b.metrics.AlertFlushTotal.Inc()
	b.metrics.AlertFlushBatchSize.Observe(float64(len(alerts)))

	batch := &pgx.Batch{}
	const insertAlert = `
		INSERT INTO alerts (id, rule_id, severity, correlation_key, triggered_at, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 'new', NOW())
		ON CONFLICT DO NOTHING
	`
	const insertAlertEvent = `
		INSERT INTO alert_events (alert_id, event_id, received_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`

	for _, alert := range alerts {
		batch.Queue(insertAlert,
			alert.ID,
			alert.RuleID,
			alert.Severity,
			alert.CorrelationKey,
			alert.TriggeredAt,
		)
		batch.Queue(insertAlertEvent,
			alert.ID,
			alert.EventID,
			alert.ReceivedAt,
		)
	}

	br := b.db.SendBatch(ctx, batch)
	defer br.Close()

	failedCount := 0
	for i := range alerts {
		for j := 0; j < 2; j++ {
			if _, err := br.Exec(); err != nil {
				failedCount++
				b.log.WithError(err).WithFields(logrus.Fields{
					"alert_id": alerts[i].ID,
					"op":       j,
				}).Warn("alert batch insert failed for individual query")
			}
		}
	}

	b.metrics.AlertBufferDepth.Sub(float64(len(alerts)))

	if failedCount > 0 {
		b.metrics.AlertFlushFailed.Inc()
		b.log.WithField("failed_count", failedCount).Warn("some alert batch inserts failed")
	}
}
