package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
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
	alertBuilder *PGAlertBuilder
	metrics      *EngineMetrics
	log          *logrus.Logger
	persistence  *persistentRuntimeStore
	degradedMode atomic.Bool

	advancedMu        sync.Mutex
	sequenceStates    map[string]*sequenceRuntimeState
	correlationStates map[string]*correlationRuntimeState
	anomalyStates     map[string]*anomalyRuntimeState
	reloadInterval    time.Duration
	reloadStopCh      chan struct{}
	reloadWG          sync.WaitGroup
	recoveryStopCh    chan struct{}
	recoveryWG        sync.WaitGroup
}

const defaultRuleReloadInterval = 30 * time.Second

type thresholdPersistedData struct {
	Count         int       `json:"count"`
	LastAlertTime time.Time `json:"last_alert_time"`
	LastEventID   string    `json:"last_event_id"`
}

type sequencePersistedData struct {
	StepIndex     int       `json:"step_index"`
	LastAlertTime time.Time `json:"last_alert_time"`
	LastEventID   string    `json:"last_event_id"`
}

type correlationPersistedData struct {
	Count         int       `json:"count"`
	DistinctTypes []string  `json:"distinct_types"`
	LastAlertTime time.Time `json:"last_alert_time"`
	LastEventID   string    `json:"last_event_id"`
}

type anomalyHistoryPersistedData struct {
	BucketStart time.Time `json:"bucket_start"`
	Count       int       `json:"count"`
}

type anomalyPersistedData struct {
	CurrentBucketStart time.Time                     `json:"current_bucket_start"`
	CurrentCount       int                           `json:"current_count"`
	History            []anomalyHistoryPersistedData `json:"history"`
	LastAlertTime      time.Time                     `json:"last_alert_time"`
	LastEventID        string                        `json:"last_event_id"`
}

func NewRuleEngine(ctx context.Context, db *pgxpool.Pool, log *logrus.Logger) (*RuleEngine, error) {
	metrics := NewEngineMetrics()

	ab := NewPGAlertBuilder(db, log, metrics)

	engine := &RuleEngine{
		loader:            NewPGRuleLoader(db),
		compiler:          NewRuleCompiler(),
		matcher:           NewRuleMatcher(),
		alertWriter:       ab,
		alertBuilder:      ab,
		metrics:           metrics,
		log:               log,
		sequenceStates:    make(map[string]*sequenceRuntimeState),
		correlationStates: make(map[string]*correlationRuntimeState),
		anomalyStates:     make(map[string]*anomalyRuntimeState),
		reloadInterval:    defaultRuleReloadInterval,
		reloadStopCh:      make(chan struct{}),
		recoveryStopCh:    make(chan struct{}),
	}
	engine.stateStore = NewStateStore(30*time.Second, 100000, log, metrics)
	engine.persistence = newPersistentRuntimeStore(db, log, metrics)

	if err := engine.ReloadRules(ctx); err != nil {
		return nil, err
	}

	if err := engine.restoreRuntimeState(ctx); err != nil {
		engine.degradedMode.Store(true)
		engine.metrics.DegradedMode.Set(1)
		engine.metrics.StateRestoreFailed.Inc()
		engine.log.WithError(err).Error("failed to restore runtime state; entering degraded mode")
		engine.startDegradedModeRecovery(ctx)
	} else {
		engine.metrics.DegradedMode.Set(0)
	}

	if err := engine.persistence.loadActiveDedupEntries(ctx); err != nil {
		engine.log.WithError(err).Warn("failed to load dedup entries from DB; in-memory dedup starts fresh")
	}

	engine.alertBuilder.Start()
	engine.persistence.Start()
	engine.startRuleReloadLoop(ctx)

	return engine, nil
}

func (e *RuleEngine) ProcessEvent(ctx context.Context, event *eventDomain.Event) error {
	if event == nil {
		return nil
	}

	e.metrics.EventsTotal.Inc()

	if e.degradedMode.Load() {
		e.metrics.EventsDegradedDroppedTotal.Inc()
		return nil
	}

	start := time.Now()
	defer func() {
		e.metrics.ProcessingLatency.Observe(time.Since(start).Seconds())
	}()

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
		receivedAt := event.ReceivedAt
		if receivedAt.IsZero() {
			receivedAt = time.Now().UTC()
		}

		correlationKey, ok := BuildCorrelationKey(rule, event, e.matcher)
		if !ok {
			if evalErr := e.persistence.recordRuleEvaluation(
				ctx,
				rule.ID,
				event.ID,
				receivedAt,
				false,
				"correlation_key_unresolved",
				"",
				map[string]any{"rule_type": rule.Type},
			); evalErr != nil {
				e.log.WithError(evalErr).WithFields(logrus.Fields{
					"rule_id":  rule.ID,
					"event_id": event.ID,
				}).Warn("failed to persist rule evaluation")
			}
			continue
		}
		stateKey := buildRuntimeStateKey(rule.ID.String(), correlationKey)

		triggeredState, triggered, reason, details := e.evaluateRule(ctx, rule, stateKey, correlationKey, event, eventType)
		if details == nil {
			details = map[string]any{}
		}
		details["rule_type"] = rule.Type
		if evalErr := e.persistence.recordRuleEvaluation(
			ctx,
			rule.ID,
			event.ID,
			receivedAt,
			triggered,
			reason,
			correlationKey,
			details,
		); evalErr != nil {
			e.log.WithError(evalErr).WithFields(logrus.Fields{
				"rule_id":         rule.ID,
				"event_id":        event.ID,
				"correlation_key": correlationKey,
				"matched":         triggered,
			}).Warn("failed to persist rule evaluation")
		}
		if !triggered {
			continue
		}

		dedupKey := buildAlertDedupKey(rule.ID.String(), correlationKey, triggeredState.LastSeen, rule.Window)
		dedupUntil := triggeredState.LastSeen
		if dedupUntil.IsZero() {
			dedupUntil = time.Now().UTC()
		}
		dedupUntil = dedupUntil.Add(rule.Window)
		acquired, dedupErr := e.persistence.tryAcquireAlertDedup(ctx, dedupKey, dedupUntil)
		if dedupErr != nil {
			e.log.WithError(dedupErr).WithFields(logrus.Fields{
				"rule_id":         rule.ID,
				"event_id":        event.ID,
				"correlation_key": correlationKey,
			}).Warn("failed to persist dedup key; suppressing alert for safety")
			continue
		}
		if !acquired {
			e.metrics.DuplicateAlerts.Inc()
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

	if err := e.persistence.saveCheckpoint(ctx, runtimeCheckpoint{LastEventID: event.ID, LastEventTime: event.EventTime}); err != nil {
		e.log.WithError(err).WithField("event_id", event.ID).Warn("failed to save engine checkpoint")
	}

	return nil
}

// evaluateRule executes one rule by type and returns synthesized state for alert payload.
func (e *RuleEngine) evaluateRule(ctx context.Context, rule CompiledRule, stateKey string, correlationKey string, event *eventDomain.Event, eventType string) (State, bool, string, map[string]any) {
	switch rule.Type {
	case "threshold":
		return e.evaluateThreshold(ctx, rule, stateKey, correlationKey, event)
	case "sequence":
		return e.evaluateSequence(ctx, rule, stateKey, correlationKey, event, eventType)
	case "correlation":
		return e.evaluateCorrelation(ctx, rule, stateKey, correlationKey, event, eventType)
	case "anomaly":
		return e.evaluateAnomaly(ctx, rule, stateKey, correlationKey, event)
	default:
		return State{}, false, "unsupported_rule_type", map[string]any{"rule_type": rule.Type}
	}
}

func (e *RuleEngine) evaluateThreshold(ctx context.Context, rule CompiledRule, stateKey string, correlationKey string, event *eventDomain.Event) (State, bool, string, map[string]any) {
	// Threshold mode keeps existing behavior for backward compatibility.
	state, canEvaluate := e.stateStore.Update(rule, stateKey, event.EventTime, event.ID)
	if !canEvaluate {
		return State{}, false, "state_limit_reached", map[string]any{
			"threshold":  rule.Threshold,
			"window_sec": int(rule.Window.Seconds()),
		}
	}

	if err := e.persistThresholdState(ctx, rule, correlationKey, state); err != nil {
		e.log.WithError(err).WithFields(logrus.Fields{"rule_id": rule.ID, "correlation_key": correlationKey}).Warn("failed to persist threshold runtime state")
	}

	if state.Count < rule.Threshold {
		return State{}, false, "threshold_not_reached", map[string]any{
			"count":      state.Count,
			"threshold":  rule.Threshold,
			"window_sec": int(rule.Window.Seconds()),
		}
	}

	if !state.LastAlertTime.IsZero() && state.LastSeen.Sub(state.LastAlertTime) < rule.Window {
		return State{}, false, "suppressed_within_window", map[string]any{
			"count":      state.Count,
			"threshold":  rule.Threshold,
			"window_sec": int(rule.Window.Seconds()),
		}
	}

	e.stateStore.MarkAlert(stateKey, state.LastSeen)
	state.LastAlertTime = state.LastSeen
	if err := e.persistThresholdState(ctx, rule, correlationKey, state); err != nil {
		e.log.WithError(err).WithFields(logrus.Fields{"rule_id": rule.ID, "correlation_key": correlationKey}).Warn("failed to persist threshold alert marker")
	}
	return state, true, "threshold_triggered", map[string]any{
		"count":      state.Count,
		"threshold":  rule.Threshold,
		"window_sec": int(rule.Window.Seconds()),
	}
}

func (e *RuleEngine) evaluateSequence(ctx context.Context, rule CompiledRule, stateKey string, correlationKey string, event *eventDomain.Event, eventType string) (State, bool, string, map[string]any) {
	// Sequence mode tracks ordered event types inside one correlation key and time window.
	now := event.EventTime
	if now.IsZero() {
		now = time.Now().UTC()
	}

	e.advancedMu.Lock()
	defer e.advancedMu.Unlock()

	state, ok := e.sequenceStates[stateKey]
	if !ok {
		state = &sequenceRuntimeState{}
		e.sequenceStates[stateKey] = state
	}

	if !state.FirstSeen.IsZero() && now.Sub(state.FirstSeen) > rule.Window {
		state.StepIndex = 0
		state.FirstSeen = time.Time{}
		state.LastSeen = now
	}

	firstStep := rule.SequenceSteps[0]
	if state.StepIndex == 0 {
		if eventType != firstStep {
			return State{}, false, "sequence_first_step_not_matched", map[string]any{
				"event_type": eventType,
				"expected":   firstStep,
			}
		}
		state.StepIndex = 1
		state.FirstSeen = now
		state.LastSeen = now
		state.LastEventID = event.ID
		e.persistSequenceState(ctx, rule, correlationKey, state, now)
		return State{}, false, "sequence_started", map[string]any{
			"event_type":  eventType,
			"step_index":  state.StepIndex,
			"total_steps": len(rule.SequenceSteps),
		}
	}

	expected := rule.SequenceSteps[state.StepIndex]
	if eventType == expected {
		state.StepIndex++
		state.LastSeen = now
		state.LastEventID = event.ID
		if state.StepIndex < len(rule.SequenceSteps) {
			e.persistSequenceState(ctx, rule, correlationKey, state, now)
			return State{}, false, "sequence_progressed", map[string]any{
				"event_type":  eventType,
				"step_index":  state.StepIndex,
				"total_steps": len(rule.SequenceSteps),
			}
		}

		if !state.LastAlertTime.IsZero() && now.Sub(state.LastAlertTime) < rule.Window {
			state.StepIndex = 0
			state.FirstSeen = time.Time{}
			return State{}, false, "suppressed_within_window", map[string]any{
				"window_sec": int(rule.Window.Seconds()),
			}
		}

		state.LastAlertTime = now
		e.persistSequenceState(ctx, rule, correlationKey, state, now)
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
		e.persistSequenceState(ctx, rule, correlationKey, state, now)
		return triggered, true, "sequence_triggered", map[string]any{
			"event_type": eventType,
			"steps":      rule.SequenceSteps,
		}
	}

	if eventType == firstStep {
		state.StepIndex = 1
		state.FirstSeen = now
		state.LastSeen = now
		state.LastEventID = event.ID
		e.persistSequenceState(ctx, rule, correlationKey, state, now)
		return State{}, false, "sequence_restarted", map[string]any{
			"event_type":  eventType,
			"step_index":  state.StepIndex,
			"total_steps": len(rule.SequenceSteps),
		}
	}

	return State{}, false, "sequence_out_of_order", map[string]any{
		"event_type": eventType,
		"expected":   expected,
	}
}

func (e *RuleEngine) evaluateCorrelation(ctx context.Context, rule CompiledRule, stateKey string, correlationKey string, event *eventDomain.Event, eventType string) (State, bool, string, map[string]any) {
	// Correlation mode triggers when volume and distinct event diversity are both satisfied.
	now := event.EventTime
	if now.IsZero() {
		now = time.Now().UTC()
	}

	e.advancedMu.Lock()
	defer e.advancedMu.Unlock()

	state, ok := e.correlationStates[stateKey]
	if !ok {
		state = &correlationRuntimeState{DistinctTypes: make(map[string]struct{})}
		e.correlationStates[stateKey] = state
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
	e.persistCorrelationState(ctx, rule, correlationKey, state, now)

	if state.Count < rule.Threshold {
		return State{}, false, "correlation_volume_not_reached", map[string]any{
			"count":        state.Count,
			"threshold":    rule.Threshold,
			"distinct":     len(state.DistinctTypes),
			"min_distinct": rule.MinDistinctEventTypes,
		}
	}
	if len(state.DistinctTypes) < rule.MinDistinctEventTypes {
		return State{}, false, "correlation_distinct_not_reached", map[string]any{
			"count":        state.Count,
			"threshold":    rule.Threshold,
			"distinct":     len(state.DistinctTypes),
			"min_distinct": rule.MinDistinctEventTypes,
		}
	}
	if !state.LastAlertTime.IsZero() && now.Sub(state.LastAlertTime) < rule.Window {
		return State{}, false, "suppressed_within_window", map[string]any{
			"window_sec": int(rule.Window.Seconds()),
		}
	}

	state.LastAlertTime = now
	e.persistCorrelationState(ctx, rule, correlationKey, state, now)
	return State{
			Count:         state.Count,
			FirstSeen:     state.FirstSeen,
			LastSeen:      state.LastSeen,
			LastAlertTime: state.LastAlertTime,
			RuleID:        rule.ID,
			LastEventID:   state.LastEventID,
		}, true, "correlation_triggered", map[string]any{
			"count":        state.Count,
			"threshold":    rule.Threshold,
			"distinct":     len(state.DistinctTypes),
			"min_distinct": rule.MinDistinctEventTypes,
		}
}

func (e *RuleEngine) evaluateAnomaly(ctx context.Context, rule CompiledRule, stateKey string, correlationKey string, event *eventDomain.Event) (State, bool, string, map[string]any) {
	// Anomaly mode compares current bucket count against baseline average * spike factor.
	now := event.EventTime
	if now.IsZero() {
		now = time.Now().UTC()
	}
	bucketStart := now.Truncate(rule.Window)

	e.advancedMu.Lock()
	defer e.advancedMu.Unlock()

	state, ok := e.anomalyStates[stateKey]
	if !ok {
		state = &anomalyRuntimeState{}
		e.anomalyStates[stateKey] = state
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
	e.persistAnomalyState(ctx, rule, correlationKey, state, now)

	if len(state.History) == 0 {
		return State{}, false, "anomaly_baseline_unavailable", map[string]any{
			"current_count": state.CurrentCount,
		}
	}

	total := 0
	for _, bucket := range state.History {
		total += bucket.Count
	}
	baselineAvg := float64(total) / float64(len(state.History))
	triggerThreshold := math.Max(float64(rule.AnomalyMinCount), baselineAvg*rule.SpikeFactor)

	if float64(state.CurrentCount) < triggerThreshold {
		return State{}, false, "anomaly_spike_not_reached", map[string]any{
			"current_count":     state.CurrentCount,
			"trigger_threshold": triggerThreshold,
			"baseline_avg":      baselineAvg,
			"spike_factor":      rule.SpikeFactor,
			"anomaly_min_count": rule.AnomalyMinCount,
		}
	}
	if !state.LastAlertTime.IsZero() && now.Sub(state.LastAlertTime) < rule.Window {
		return State{}, false, "suppressed_within_window", map[string]any{
			"window_sec": int(rule.Window.Seconds()),
		}
	}

	state.LastAlertTime = now
	e.persistAnomalyState(ctx, rule, correlationKey, state, now)
	return State{
			Count:         state.CurrentCount,
			FirstSeen:     state.CurrentBucketStart,
			LastSeen:      state.LastSeen,
			LastAlertTime: state.LastAlertTime,
			RuleID:        rule.ID,
			LastEventID:   state.LastEventID,
		}, true, "anomaly_triggered", map[string]any{
			"current_count":     state.CurrentCount,
			"trigger_threshold": triggerThreshold,
			"baseline_avg":      baselineAvg,
			"spike_factor":      rule.SpikeFactor,
			"anomaly_min_count": rule.AnomalyMinCount,
		}
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
	if e.reloadStopCh != nil {
		close(e.reloadStopCh)
		e.reloadWG.Wait()
	}
	if e.recoveryStopCh != nil {
		close(e.recoveryStopCh)
		e.recoveryWG.Wait()
	}
	e.persistence.Stop()
	e.alertBuilder.Stop()
	e.stateStore.Stop()
}

func (e *RuleEngine) startRuleReloadLoop(ctx context.Context) {
	if e.reloadInterval <= 0 {
		return
	}

	e.reloadWG.Add(1)
	go func() {
		defer e.reloadWG.Done()

		ticker := time.NewTicker(e.reloadInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := e.ReloadRules(ctx); err != nil {
					e.log.WithError(err).Warn("failed periodic rule reload")
				}
			case <-e.reloadStopCh:
				return
			}
		}
	}()
}

const defaultDegradedRecoveryInterval = 60 * time.Second

func (e *RuleEngine) startDegradedModeRecovery(ctx context.Context) {
	e.recoveryWG.Add(1)
	go func() {
		defer e.recoveryWG.Done()

		ticker := time.NewTicker(defaultDegradedRecoveryInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !e.degradedMode.Load() {
					return
				}

				e.metrics.DegradedRecoveryTotal.Inc()

				if err := e.restoreRuntimeState(ctx); err != nil {
					e.metrics.DegradedRecoveryFailed.Inc()
					e.log.WithError(err).Warn("degraded mode recovery attempt failed, will retry")
					continue
				}

				e.degradedMode.Store(false)
				e.metrics.DegradedMode.Set(0)
				e.log.Info("engine recovered from degraded mode, rule processing resumed")
				return
			case <-e.recoveryStopCh:
				return
			}
		}
	}()
}

func buildRuntimeStateKey(ruleID string, correlationKey string) string {
	return ruleID + ":" + correlationKey
}

func buildAlertDedupKey(ruleID string, correlationKey string, triggeredAt time.Time, window time.Duration) string {
	if triggeredAt.IsZero() {
		triggeredAt = time.Now().UTC()
	}
	bucket := triggeredAt.Unix()
	if window > 0 {
		bucket = triggeredAt.Truncate(window).Unix()
	}
	return fmt.Sprintf("%s|%s|%d", ruleID, correlationKey, bucket)
}

func (e *RuleEngine) restoreRuntimeState(ctx context.Context) error {
	e.metrics.StateRestoreTotal.Inc()
	start := time.Now()

	if err := e.persistence.pruneExpiredStates(ctx); err != nil {
		return err
	}

	rows, err := e.persistence.loadActiveStates(ctx)
	if err != nil {
		return err
	}

	runtimeRules := e.runtimeRules.Load()
	if runtimeRules == nil {
		return nil
	}

	ruleByID := make(map[string]CompiledRule)
	for _, compiledRules := range runtimeRules.(*RuntimeRules).rulesByEventType {
		for _, rule := range compiledRules {
			ruleByID[rule.ID.String()] = rule
		}
	}

	now := time.Now().UTC()
	maxStaleness := 0.0

	for _, row := range rows {
		rule, ok := ruleByID[row.RuleID.String()]
		if !ok {
			continue
		}

		staleness := now.Sub(row.LastSeenAt).Seconds()
		if staleness > maxStaleness {
			maxStaleness = staleness
		}

		switch row.StateType {
		case "threshold":
			if err := e.restoreThresholdState(rule, row); err != nil {
				e.log.WithError(err).WithFields(logrus.Fields{"rule_id": row.RuleID, "correlation_key": row.CorrelationKey}).Warn("skipping invalid threshold state row")
			}
		case "sequence":
			if err := e.restoreSequenceState(rule, row); err != nil {
				e.log.WithError(err).WithFields(logrus.Fields{"rule_id": row.RuleID, "correlation_key": row.CorrelationKey}).Warn("skipping invalid sequence state row")
			}
		case "correlation":
			if err := e.restoreCorrelationState(rule, row); err != nil {
				e.log.WithError(err).WithFields(logrus.Fields{"rule_id": row.RuleID, "correlation_key": row.CorrelationKey}).Warn("skipping invalid correlation state row")
			}
		case "anomaly":
			if err := e.restoreAnomalyState(rule, row); err != nil {
				e.log.WithError(err).WithFields(logrus.Fields{"rule_id": row.RuleID, "correlation_key": row.CorrelationKey}).Warn("skipping invalid anomaly state row")
			}
		default:
			e.log.WithFields(logrus.Fields{"rule_id": row.RuleID, "correlation_key": row.CorrelationKey, "state_type": row.StateType}).Warn("unknown persisted state type")
		}
	}

	e.metrics.StateStaleness.Set(maxStaleness)
	e.metrics.StateRestoreLatency.Observe(time.Since(start).Seconds())

	if checkpoint, found, checkpointErr := e.persistence.loadCheckpoint(ctx); checkpointErr != nil {
		e.log.WithError(checkpointErr).Warn("failed to load engine checkpoint")
	} else if found {
		e.log.WithFields(logrus.Fields{
			"last_event_id":   checkpoint.LastEventID,
			"last_event_time": checkpoint.LastEventTime,
		}).Info("engine checkpoint restored")
	}

	return nil
}

func (e *RuleEngine) restoreThresholdState(rule CompiledRule, row persistedRuntimeStateRow) error {
	var payload thresholdPersistedData
	if err := json.Unmarshal(row.StateData, &payload); err != nil {
		return err
	}

	lastEventID, _ := uuidFromString(payload.LastEventID)
	state := State{
		Count:         payload.Count,
		FirstSeen:     row.FirstSeenAt,
		LastSeen:      row.LastSeenAt,
		ExpiresAt:     row.ExpiresAt,
		LastAlertTime: payload.LastAlertTime,
		RuleID:        rule.ID,
		LastEventID:   lastEventID,
	}

	stateKey := buildRuntimeStateKey(rule.ID.String(), row.CorrelationKey)
	e.stateStore.Restore(stateKey, state)
	return nil
}

func (e *RuleEngine) restoreSequenceState(rule CompiledRule, row persistedRuntimeStateRow) error {
	var payload sequencePersistedData
	if err := json.Unmarshal(row.StateData, &payload); err != nil {
		return err
	}
	lastEventID, _ := uuidFromString(payload.LastEventID)

	e.advancedMu.Lock()
	e.sequenceStates[buildRuntimeStateKey(rule.ID.String(), row.CorrelationKey)] = &sequenceRuntimeState{
		StepIndex:     payload.StepIndex,
		FirstSeen:     row.FirstSeenAt,
		LastSeen:      row.LastSeenAt,
		LastAlertTime: payload.LastAlertTime,
		LastEventID:   lastEventID,
	}
	e.advancedMu.Unlock()
	return nil
}

func (e *RuleEngine) restoreCorrelationState(rule CompiledRule, row persistedRuntimeStateRow) error {
	var payload correlationPersistedData
	if err := json.Unmarshal(row.StateData, &payload); err != nil {
		return err
	}

	distinct := make(map[string]struct{}, len(payload.DistinctTypes))
	for _, eventType := range payload.DistinctTypes {
		distinct[eventType] = struct{}{}
	}
	lastEventID, _ := uuidFromString(payload.LastEventID)

	e.advancedMu.Lock()
	e.correlationStates[buildRuntimeStateKey(rule.ID.String(), row.CorrelationKey)] = &correlationRuntimeState{
		Count:         payload.Count,
		DistinctTypes: distinct,
		FirstSeen:     row.FirstSeenAt,
		LastSeen:      row.LastSeenAt,
		LastAlertTime: payload.LastAlertTime,
		LastEventID:   lastEventID,
	}
	e.advancedMu.Unlock()
	return nil
}

func (e *RuleEngine) restoreAnomalyState(rule CompiledRule, row persistedRuntimeStateRow) error {
	var payload anomalyPersistedData
	if err := json.Unmarshal(row.StateData, &payload); err != nil {
		return err
	}

	history := make([]anomalyBucket, 0, len(payload.History))
	for _, item := range payload.History {
		history = append(history, anomalyBucket{BucketStart: item.BucketStart, Count: item.Count})
	}
	lastEventID, _ := uuidFromString(payload.LastEventID)

	e.advancedMu.Lock()
	e.anomalyStates[buildRuntimeStateKey(rule.ID.String(), row.CorrelationKey)] = &anomalyRuntimeState{
		CurrentBucketStart: payload.CurrentBucketStart,
		CurrentCount:       payload.CurrentCount,
		History:            history,
		LastSeen:           row.LastSeenAt,
		LastAlertTime:      payload.LastAlertTime,
		LastEventID:        lastEventID,
	}
	e.advancedMu.Unlock()
	return nil
}

func (e *RuleEngine) persistThresholdState(ctx context.Context, rule CompiledRule, correlationKey string, state State) error {
	return e.persistence.upsertState(
		ctx,
		rule.ID,
		correlationKey,
		"threshold",
		thresholdPersistedData{
			Count:         state.Count,
			LastAlertTime: state.LastAlertTime,
			LastEventID:   state.LastEventID.String(),
		},
		state.FirstSeen,
		state.LastSeen,
		state.ExpiresAt,
	)
}

func (e *RuleEngine) persistSequenceState(ctx context.Context, rule CompiledRule, correlationKey string, state *sequenceRuntimeState, now time.Time) {
	firstSeen := state.FirstSeen
	if firstSeen.IsZero() {
		firstSeen = now
	}
	expiresAt := firstSeen.Add(rule.Window)
	if state.StepIndex == 0 {
		expiresAt = now.Add(rule.Window)
	}
	if err := e.persistence.upsertState(
		ctx,
		rule.ID,
		correlationKey,
		"sequence",
		sequencePersistedData{
			StepIndex:     state.StepIndex,
			LastAlertTime: state.LastAlertTime,
			LastEventID:   state.LastEventID.String(),
		},
		firstSeen,
		state.LastSeen,
		expiresAt,
	); err != nil {
		e.log.WithError(err).WithFields(logrus.Fields{"rule_id": rule.ID, "correlation_key": correlationKey}).Warn("failed to persist sequence runtime state")
	}
}

func (e *RuleEngine) persistCorrelationState(ctx context.Context, rule CompiledRule, correlationKey string, state *correlationRuntimeState, now time.Time) {
	distinct := make([]string, 0, len(state.DistinctTypes))
	for eventType := range state.DistinctTypes {
		distinct = append(distinct, eventType)
	}
	firstSeen := state.FirstSeen
	if firstSeen.IsZero() {
		firstSeen = now
	}
	if err := e.persistence.upsertState(
		ctx,
		rule.ID,
		correlationKey,
		"correlation",
		correlationPersistedData{
			Count:         state.Count,
			DistinctTypes: distinct,
			LastAlertTime: state.LastAlertTime,
			LastEventID:   state.LastEventID.String(),
		},
		firstSeen,
		state.LastSeen,
		firstSeen.Add(rule.Window),
	); err != nil {
		e.log.WithError(err).WithFields(logrus.Fields{"rule_id": rule.ID, "correlation_key": correlationKey}).Warn("failed to persist correlation runtime state")
	}
}

func (e *RuleEngine) persistAnomalyState(ctx context.Context, rule CompiledRule, correlationKey string, state *anomalyRuntimeState, now time.Time) {
	history := make([]anomalyHistoryPersistedData, 0, len(state.History))
	for _, bucket := range state.History {
		history = append(history, anomalyHistoryPersistedData{BucketStart: bucket.BucketStart, Count: bucket.Count})
	}
	firstSeen := state.CurrentBucketStart
	if firstSeen.IsZero() {
		firstSeen = now
	}
	if err := e.persistence.upsertState(
		ctx,
		rule.ID,
		correlationKey,
		"anomaly",
		anomalyPersistedData{
			CurrentBucketStart: state.CurrentBucketStart,
			CurrentCount:       state.CurrentCount,
			History:            history,
			LastAlertTime:      state.LastAlertTime,
			LastEventID:        state.LastEventID.String(),
		},
		firstSeen,
		state.LastSeen,
		now.Add(rule.BaselineWindow),
	); err != nil {
		e.log.WithError(err).WithFields(logrus.Fields{"rule_id": rule.ID, "correlation_key": correlationKey}).Warn("failed to persist anomaly runtime state")
	}
}

func uuidFromString(value string) (uuid.UUID, bool) {
	if value == "" {
		return uuid.Nil, false
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, false
	}
	return parsed, true
}
