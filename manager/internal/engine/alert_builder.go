package engine

import (
	"context"
	"time"

	"github.com/google/uuid"
	eventDomain "github.com/hildanku/xemarify/internal/modules/event/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	db *pgxpool.Pool
}

func NewPGAlertBuilder(db *pgxpool.Pool) *PGAlertBuilder {
	return &PGAlertBuilder{db: db}
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
	tx, err := b.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const insertAlert = `
		INSERT INTO alerts (id, rule_id, severity, correlation_key, triggered_at, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 'new', NOW())
	`

	if _, err := tx.Exec(ctx, insertAlert,
		alert.ID,
		alert.RuleID,
		alert.Severity,
		alert.CorrelationKey,
		alert.TriggeredAt,
	); err != nil {
		return err
	}

	const insertAlertEvent = `
		INSERT INTO alert_events (alert_id, event_id, received_at)
		VALUES ($1, $2, $3)
	`

	if _, err := tx.Exec(ctx, insertAlertEvent,
		alert.ID,
		alert.EventID,
		alert.ReceivedAt,
	); err != nil {
		return err
	}

	// Keep a lightweight evaluation trail for triggered matches.
	const insertRuleEvaluation = `
		INSERT INTO rule_evaluations (rule_id, event_id, received_at, created_at)
		VALUES ($1, $2, $3, NOW())
	`

	if _, err := tx.Exec(ctx, insertRuleEvaluation,
		alert.RuleID,
		alert.EventID,
		alert.ReceivedAt,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
