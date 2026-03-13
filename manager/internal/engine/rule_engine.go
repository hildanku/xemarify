package engine

import (
	"context"
	"sync/atomic"
	"time"

	eventDomain "github.com/hildanku/xemarify/internal/modules/event/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

type Engine interface {
	ProcessEvent(ctx context.Context, event *eventDomain.Event) error
	ReloadRules(ctx context.Context) error
	Stop()
}

type RuntimeRules struct {
	rulesByEventType map[string][]CompiledRule
}

// RuleEngine implements an in-memory threshold detector:
// Event -> Rule Match -> State Update -> Threshold Check -> Alert.
type RuleEngine struct {
	runtimeRules atomic.Value
	loader       RuleLoader
	compiler     *RuleCompiler
	matcher      *RuleMatcher
	stateStore   *StateStore
	alertWriter  AlertWriter
	metrics      *EngineMetrics
	log          *logrus.Logger
}

func NewRuleEngine(ctx context.Context, db *pgxpool.Pool, log *logrus.Logger) (*RuleEngine, error) {
	metrics := NewEngineMetrics()

	engine := &RuleEngine{
		loader:      NewPGRuleLoader(db),
		compiler:    NewRuleCompiler(),
		matcher:     NewRuleMatcher(),
		alertWriter: NewPGAlertBuilder(db),
		metrics:     metrics,
		log:         log,
	}
	engine.stateStore = NewStateStore(30*time.Second, 100000, log, metrics)

	if err := engine.ReloadRules(ctx); err != nil {
		return nil, err
	}

	return engine, nil
}

func (e *RuleEngine) ProcessEvent(ctx context.Context, event *eventDomain.Event) error {
	if event == nil {
		return nil
	}

	start := time.Now()
	defer func() {
		e.metrics.ProcessingLatency.Observe(time.Since(start).Seconds())
	}()
	e.metrics.EventsTotal.Inc()

	eventType := e.matcher.EventType(event)
	if eventType == "" {
		return nil
	}

	runtimeRules := e.runtimeRules.Load()
	if runtimeRules == nil {
		return nil
	}

	rules := runtimeRules.(*RuntimeRules).rulesByEventType[eventType]
	if len(rules) == 0 {
		return nil
	}
	e.metrics.RulesEvaluatedTotal.Add(float64(len(rules)))

	for _, rule := range rules {
		correlationKey, ok := BuildCorrelationKey(rule, event, e.matcher)
		if !ok {
			continue
		}

		state, canEvaluate := e.stateStore.Update(rule, correlationKey, event.EventTime, event.ID)
		if !canEvaluate {
			continue
		}
		if state.Count < rule.Threshold {
			continue
		}

		if !state.LastAlertTime.IsZero() && state.LastSeen.Sub(state.LastAlertTime) < rule.Window {
			continue
		}

		alert := e.alertWriter.Build(rule, correlationKey, state, event)
		if err := e.alertWriter.Persist(ctx, alert); err != nil {
			e.log.WithError(err).WithFields(logrus.Fields{
				"rule_id":         rule.ID,
				"event_id":        event.ID,
				"correlation_key": correlationKey,
			}).Error("failed to persist detection alert")
			continue
		}

		e.log.WithFields(logrus.Fields{
			"rule_id":         rule.ID,
			"event_id":        event.ID,
			"correlation_key": correlationKey,
			"count":           state.Count,
			"threshold":       rule.Threshold,
		}).Info("detection alert triggered")
		e.metrics.AlertsTotal.Inc()

		e.stateStore.MarkAlert(correlationKey, state.LastSeen)
	}

	return nil
}

func (e *RuleEngine) ReloadRules(ctx context.Context) error {
	storedRules, err := e.loader.LoadEnabledRules(ctx)
	if err != nil {
		return err
	}

	indexedRules, compileErrs := e.compiler.Compile(storedRules)
	for _, compileErr := range compileErrs {
		e.log.WithError(compileErr).Warn("skipping invalid detection rule")
	}

	e.runtimeRules.Store(&RuntimeRules{rulesByEventType: indexedRules})

	e.log.WithFields(logrus.Fields{
		"rules_loaded":  len(storedRules),
		"event_types":   len(indexedRules),
		"rules_skipped": len(compileErrs),
	}).Info("rule engine rules reloaded")

	return nil
}

func (e *RuleEngine) Stop() {
	e.stateStore.Stop()
}
