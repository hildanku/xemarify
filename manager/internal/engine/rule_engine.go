package engine

import (
	"context"
	"time"

	eventDomain "github.com/hildanku/xemarify/internal/modules/event/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

type Engine interface {
	ProcessEvent(ctx context.Context, event *eventDomain.Event) error
	Stop()
}

// RuleEngine implements an in-memory threshold detector:
// Event -> Rule Match -> State Update -> Threshold Check -> Alert.
type RuleEngine struct {
	rulesByEventType map[string][]CompiledRule
	matcher          *RuleMatcher
	stateStore       *StateStore
	alertWriter      AlertWriter
	log              *logrus.Logger
}

func NewRuleEngine(ctx context.Context, db *pgxpool.Pool, log *logrus.Logger) (*RuleEngine, error) {
	loader := NewPGRuleLoader(db)
	storedRules, err := loader.LoadEnabledRules(ctx)
	if err != nil {
		return nil, err
	}

	compiler := NewRuleCompiler()
	indexedRules, compileErrs := compiler.Compile(storedRules)
	for _, compileErr := range compileErrs {
		log.WithError(compileErr).Warn("skipping invalid detection rule")
	}

	engine := &RuleEngine{
		rulesByEventType: indexedRules,
		matcher:          NewRuleMatcher(),
		stateStore:       NewStateStore(30 * time.Second),
		alertWriter:      NewPGAlertBuilder(db),
		log:              log,
	}

	log.WithFields(logrus.Fields{
		"rules_loaded":  len(storedRules),
		"event_types":   len(indexedRules),
		"rules_skipped": len(compileErrs),
	}).Info("rule engine initialized")

	return engine, nil
}

func (e *RuleEngine) ProcessEvent(ctx context.Context, event *eventDomain.Event) error {
	if event == nil {
		return nil
	}

	eventType := e.matcher.EventType(event)
	if eventType == "" {
		return nil
	}

	rules := e.rulesByEventType[eventType]
	if len(rules) == 0 {
		return nil
	}

	for _, rule := range rules {
		correlationKey, ok := BuildCorrelationKey(rule, event, e.matcher)
		if !ok {
			continue
		}

		state := e.stateStore.Update(rule, correlationKey, event.EventTime, event.ID)
		if state.Count < rule.Threshold {
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

		// Reset after trigger to avoid generating one alert for every subsequent event.
		e.stateStore.Reset(correlationKey)
	}

	return nil
}

func (e *RuleEngine) Stop() {
	e.stateStore.Stop()
}
