package engine

import (
	"context"
	"math"
	"sync"
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

	advancedMu        sync.Mutex
	sequenceStates    map[string]*sequenceRuntimeState
	correlationStates map[string]*correlationRuntimeState
	anomalyStates     map[string]*anomalyRuntimeState
}

func NewRuleEngine(ctx context.Context, db *pgxpool.Pool, log *logrus.Logger) (*RuleEngine, error) {
	metrics := NewEngineMetrics()

	engine := &RuleEngine{
		loader:            NewPGRuleLoader(db),
		compiler:          NewRuleCompiler(),
		matcher:           NewRuleMatcher(),
		alertWriter:       NewPGAlertBuilder(db),
		metrics:           metrics,
		log:               log,
		sequenceStates:    make(map[string]*sequenceRuntimeState),
		correlationStates: make(map[string]*correlationRuntimeState),
		anomalyStates:     make(map[string]*anomalyRuntimeState),
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

		triggeredState, triggered := e.evaluateRule(rule, correlationKey, event, eventType)
		if !triggered {
			continue
		}

		alert := e.alertWriter.Build(rule, correlationKey, triggeredState, event)
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
			"count":           triggeredState.Count,
			"rule_type":       rule.Type,
		}).Info("detection alert triggered")
		e.metrics.AlertsTotal.Inc()
	}

	return nil
}

// evaluateRule executes one rule by type and returns synthesized state for alert payload.
func (e *RuleEngine) evaluateRule(rule CompiledRule, correlationKey string, event *eventDomain.Event, eventType string) (State, bool) {
	switch rule.Type {
	case "threshold":
		return e.evaluateThreshold(rule, correlationKey, event)
	case "sequence":
		return e.evaluateSequence(rule, correlationKey, event, eventType)
	case "correlation":
		return e.evaluateCorrelation(rule, correlationKey, event, eventType)
	case "anomaly":
		return e.evaluateAnomaly(rule, correlationKey, event)
	default:
		return State{}, false
	}
}

func (e *RuleEngine) evaluateThreshold(rule CompiledRule, correlationKey string, event *eventDomain.Event) (State, bool) {
	// Threshold mode keeps existing behavior for backward compatibility.
	state, canEvaluate := e.stateStore.Update(rule, correlationKey, event.EventTime, event.ID)
	if !canEvaluate {
		return State{}, false
	}
	if state.Count < rule.Threshold {
		return State{}, false
	}

	if !state.LastAlertTime.IsZero() && state.LastSeen.Sub(state.LastAlertTime) < rule.Window {
		return State{}, false
	}

	e.stateStore.MarkAlert(correlationKey, state.LastSeen)
	return state, true
}

func (e *RuleEngine) evaluateSequence(rule CompiledRule, correlationKey string, event *eventDomain.Event, eventType string) (State, bool) {
	// Sequence mode tracks ordered event types inside one correlation key and time window.
	now := event.EventTime
	if now.IsZero() {
		now = time.Now().UTC()
	}

	e.advancedMu.Lock()
	defer e.advancedMu.Unlock()

	state, ok := e.sequenceStates[correlationKey]
	if !ok {
		state = &sequenceRuntimeState{}
		e.sequenceStates[correlationKey] = state
	}

	if !state.FirstSeen.IsZero() && now.Sub(state.FirstSeen) > rule.Window {
		state.StepIndex = 0
		state.FirstSeen = time.Time{}
	}

	firstStep := rule.SequenceSteps[0]
	if state.StepIndex == 0 {
		if eventType != firstStep {
			return State{}, false
		}
		state.StepIndex = 1
		state.FirstSeen = now
		state.LastSeen = now
		state.LastEventID = event.ID
		return State{}, false
	}

	expected := rule.SequenceSteps[state.StepIndex]
	if eventType == expected {
		state.StepIndex++
		state.LastSeen = now
		state.LastEventID = event.ID
		if state.StepIndex < len(rule.SequenceSteps) {
			return State{}, false
		}

		if !state.LastAlertTime.IsZero() && now.Sub(state.LastAlertTime) < rule.Window {
			state.StepIndex = 0
			state.FirstSeen = time.Time{}
			return State{}, false
		}

		state.LastAlertTime = now
		triggered := State{
			Count:         len(rule.SequenceSteps),
			FirstSeen:     state.FirstSeen,
			LastSeen:      now,
			LastAlertTime: now,
			RuleID:        rule.ID,
			LastEventID:   event.ID,
		}
		state.StepIndex = 0
		state.FirstSeen = time.Time{}
		return triggered, true
	}

	if eventType == firstStep {
		state.StepIndex = 1
		state.FirstSeen = now
		state.LastSeen = now
		state.LastEventID = event.ID
	}

	return State{}, false
}

func (e *RuleEngine) evaluateCorrelation(rule CompiledRule, correlationKey string, event *eventDomain.Event, eventType string) (State, bool) {
	// Correlation mode triggers when volume and distinct event diversity are both satisfied.
	now := event.EventTime
	if now.IsZero() {
		now = time.Now().UTC()
	}

	e.advancedMu.Lock()
	defer e.advancedMu.Unlock()

	state, ok := e.correlationStates[correlationKey]
	if !ok {
		state = &correlationRuntimeState{DistinctTypes: make(map[string]struct{})}
		e.correlationStates[correlationKey] = state
	}

	if !state.FirstSeen.IsZero() && now.Sub(state.FirstSeen) > rule.Window {
		state.Count = 0
		state.FirstSeen = time.Time{}
		state.DistinctTypes = make(map[string]struct{})
	}

	if state.FirstSeen.IsZero() {
		state.FirstSeen = now
	}

	state.Count++
	state.LastSeen = now
	state.LastEventID = event.ID
	state.DistinctTypes[eventType] = struct{}{}

	if state.Count < rule.Threshold {
		return State{}, false
	}
	if len(state.DistinctTypes) < rule.MinDistinctEventTypes {
		return State{}, false
	}
	if !state.LastAlertTime.IsZero() && now.Sub(state.LastAlertTime) < rule.Window {
		return State{}, false
	}

	state.LastAlertTime = now
	return State{
		Count:         state.Count,
		FirstSeen:     state.FirstSeen,
		LastSeen:      state.LastSeen,
		LastAlertTime: state.LastAlertTime,
		RuleID:        rule.ID,
		LastEventID:   state.LastEventID,
	}, true
}

func (e *RuleEngine) evaluateAnomaly(rule CompiledRule, correlationKey string, event *eventDomain.Event) (State, bool) {
	// Anomaly mode compares current bucket count against baseline average * spike factor.
	now := event.EventTime
	if now.IsZero() {
		now = time.Now().UTC()
	}
	bucketStart := now.Truncate(rule.Window)

	e.advancedMu.Lock()
	defer e.advancedMu.Unlock()

	state, ok := e.anomalyStates[correlationKey]
	if !ok {
		state = &anomalyRuntimeState{}
		e.anomalyStates[correlationKey] = state
	}

	if state.CurrentBucketStart.IsZero() {
		state.CurrentBucketStart = bucketStart
	}

	if !state.CurrentBucketStart.Equal(bucketStart) {
		state.History = append(state.History, anomalyBucket{BucketStart: state.CurrentBucketStart, Count: state.CurrentCount})
		state.CurrentBucketStart = bucketStart
		state.CurrentCount = 0
	}

	baselineCutoff := now.Add(-rule.BaselineWindow)
	filtered := make([]anomalyBucket, 0, len(state.History))
	for _, bucket := range state.History {
		if bucket.BucketStart.After(baselineCutoff) {
			filtered = append(filtered, bucket)
		}
	}
	state.History = filtered

	state.CurrentCount++
	state.LastSeen = now
	state.LastEventID = event.ID

	if len(state.History) == 0 {
		return State{}, false
	}

	total := 0
	for _, bucket := range state.History {
		total += bucket.Count
	}
	baselineAvg := float64(total) / float64(len(state.History))
	triggerThreshold := math.Max(float64(rule.AnomalyMinCount), baselineAvg*rule.SpikeFactor)

	if float64(state.CurrentCount) < triggerThreshold {
		return State{}, false
	}
	if !state.LastAlertTime.IsZero() && now.Sub(state.LastAlertTime) < rule.Window {
		return State{}, false
	}

	state.LastAlertTime = now
	return State{
		Count:         state.CurrentCount,
		FirstSeen:     state.CurrentBucketStart,
		LastSeen:      state.LastSeen,
		LastAlertTime: state.LastAlertTime,
		RuleID:        rule.ID,
		LastEventID:   state.LastEventID,
	}, true
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
