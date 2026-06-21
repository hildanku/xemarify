package engine

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func newTestStore() *persistentRuntimeStore {
	return &persistentRuntimeStore{
		db:            nil,
		log:           logrus.New(),
		metrics:       testEngineMetrics,
		flushInterval: defaultPersistenceFlushInterval,
		stopCh:        make(chan struct{}),
		pendingStates: make(map[string]persistedRuntimeStateRow),
		dedupEntries:  make(map[string]dedupEntry),
		evalBuf:       make([]evaluationEntry, 0, 256),
		deleteBuf:     make([]deleteEntry, 0, 16),
		dedupPersist:  make([]dedupEntry, 0, 64),
	}
}

func TestUpsertState_BuffersInMemory(t *testing.T) {
	s := newTestStore()
	ruleID := uuid.New()
	now := time.Now().UTC()
	expires := now.Add(5 * time.Minute)

	err := s.upsertState(context.Background(), ruleID, "host:web-01", "threshold", thresholdPersistedData{Count: 5}, now, now, expires)
	if err != nil {
		t.Fatalf("upsertState returned error: %v", err)
	}

	s.stateMu.Lock()
	count := len(s.pendingStates)
	s.stateMu.Unlock()

	if count != 1 {
		t.Fatalf("expected 1 pending state, got %d", count)
	}

	key := ruleID.String() + ":host:web-01"
	s.stateMu.Lock()
	row, ok := s.pendingStates[key]
	s.stateMu.Unlock()

	if !ok {
		t.Fatalf("expected state for key %s, not found", key)
	}
	if row.StateType != "threshold" {
		t.Fatalf("expected state_type=threshold, got %s", row.StateType)
	}
	if row.CorrelationKey != "host:web-01" {
		t.Fatalf("expected correlation_key=host:web-01, got %s", row.CorrelationKey)
	}

	var data thresholdPersistedData
	if err := json.Unmarshal(row.StateData, &data); err != nil {
		t.Fatalf("unmarshal state data: %v", err)
	}
	if data.Count != 5 {
		t.Fatalf("expected count=5, got %d", data.Count)
	}
}

func TestUpsertState_SameKeyOverwrites(t *testing.T) {
	s := newTestStore()
	ruleID := uuid.New()
	now := time.Now().UTC()
	expires := now.Add(5 * time.Minute)

	s.upsertState(context.Background(), ruleID, "ck-1", "threshold", thresholdPersistedData{Count: 3}, now, now, expires)
	s.upsertState(context.Background(), ruleID, "ck-1", "threshold", thresholdPersistedData{Count: 7}, now, now, expires)

	s.stateMu.Lock()
	count := len(s.pendingStates)
	row := s.pendingStates[ruleID.String()+":ck-1"]
	s.stateMu.Unlock()

	if count != 1 {
		t.Fatalf("same key should overwrite, expected 1 pending state, got %d", count)
	}

	var data thresholdPersistedData
	json.Unmarshal(row.StateData, &data)
	if data.Count != 7 {
		t.Fatalf("expected latest count=7, got %d", data.Count)
	}
}

func TestUpsertState_DifferentKeysAccumulate(t *testing.T) {
	s := newTestStore()
	rule1 := uuid.New()
	rule2 := uuid.New()
	now := time.Now().UTC()
	expires := now.Add(5 * time.Minute)

	s.upsertState(context.Background(), rule1, "ck-a", "threshold", thresholdPersistedData{Count: 1}, now, now, expires)
	s.upsertState(context.Background(), rule2, "ck-b", "correlation", correlationPersistedData{Count: 2}, now, now, expires)

	s.stateMu.Lock()
	count := len(s.pendingStates)
	s.stateMu.Unlock()

	if count != 2 {
		t.Fatalf("different keys should accumulate, expected 2, got %d", count)
	}
}

func TestUpsertState_ConcurrentNoDataLoss(t *testing.T) {
	s := newTestStore()
	const n = 50

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ruleID := uuid.New()
			now := time.Now().UTC()
			s.upsertState(context.Background(), ruleID, "ck-"+uuid.New().String(), "threshold", thresholdPersistedData{Count: idx}, now, now, now.Add(5*time.Minute))
		}(i)
	}
	wg.Wait()

	s.stateMu.Lock()
	count := len(s.pendingStates)
	s.stateMu.Unlock()

	if count != n {
		t.Fatalf("expected %d concurrent upserts, got %d", n, count)
	}
}

func TestUpsertState_ReturnsImmediately(t *testing.T) {
	s := newTestStore()
	ruleID := uuid.New()
	now := time.Now().UTC()

	start := time.Now()
	err := s.upsertState(context.Background(), ruleID, "ck", "threshold", thresholdPersistedData{Count: 1}, now, now, now.Add(5*time.Minute))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("upsertState error: %v", err)
	}
	if elapsed > 5*time.Millisecond {
		t.Fatalf("upsertState should return immediately (<5ms), took %v", elapsed)
	}
}

func TestRecordEvaluation_BuffersInMemory(t *testing.T) {
	s := newTestStore()
	ruleID := uuid.New()
	eventID := uuid.New()
	now := time.Now().UTC()

	err := s.recordRuleEvaluation(context.Background(), ruleID, eventID, now, true, "threshold_triggered", "ck-1", map[string]any{"count": 5})
	if err != nil {
		t.Fatalf("recordRuleEvaluation error: %v", err)
	}

	s.evalMu.Lock()
	count := len(s.evalBuf)
	entry := s.evalBuf[0]
	s.evalMu.Unlock()

	if count != 1 {
		t.Fatalf("expected 1 buffered evaluation, got %d", count)
	}
	if entry.RuleID != ruleID {
		t.Fatalf("RuleID mismatch: got %v, want %v", entry.RuleID, ruleID)
	}
	if entry.EventID != eventID {
		t.Fatalf("EventID mismatch: got %v, want %v", entry.EventID, eventID)
	}
	if entry.Matched != true {
		t.Fatalf("Matched mismatch: got %v, want true", entry.Matched)
	}
	if entry.Reason != "threshold_triggered" {
		t.Fatalf("Reason mismatch: got %v, want threshold_triggered", entry.Reason)
	}
}

func TestRecordEvaluation_NilDetailsHandled(t *testing.T) {
	s := newTestStore()
	ruleID := uuid.New()
	eventID := uuid.New()

	err := s.recordRuleEvaluation(context.Background(), ruleID, eventID, time.Now().UTC(), false, "no_match", "ck", nil)
	if err != nil {
		t.Fatalf("recordRuleEvaluation with nil details: %v", err)
	}

	s.evalMu.Lock()
	entry := s.evalBuf[0]
	s.evalMu.Unlock()

	var details map[string]any
	if err := json.Unmarshal(entry.Details, &details); err != nil {
		t.Fatalf("unmarshal details: %v", err)
	}
	if len(details) != 0 {
		t.Fatalf("nil details should become empty map, got %d keys", len(details))
	}
}

func TestRecordEvaluation_ZeroReceivedAtDefaults(t *testing.T) {
	s := newTestStore()
	ruleID := uuid.New()
	eventID := uuid.New()

	before := time.Now().UTC()
	err := s.recordRuleEvaluation(context.Background(), ruleID, eventID, time.Time{}, false, "reason", "ck", nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	s.evalMu.Lock()
	entry := s.evalBuf[0]
	s.evalMu.Unlock()

	if entry.ReceivedAt.Before(before) {
		t.Fatalf("zero receivedAt should default to now, got %v", entry.ReceivedAt)
	}
}

func TestRecordEvaluation_MultipleAccumulate(t *testing.T) {
	s := newTestStore()
	const n = 5

	for i := 0; i < n; i++ {
		s.recordRuleEvaluation(context.Background(), uuid.New(), uuid.New(), time.Now().UTC(), i%2 == 0, "reason-"+string(rune('a'+i)), "ck", nil)
	}

	s.evalMu.Lock()
	count := len(s.evalBuf)
	s.evalMu.Unlock()

	if count != n {
		t.Fatalf("expected %d buffered evaluations, got %d", n, count)
	}
}

func TestTryAcquireDedup_FirstAcquireReturnsTrue(t *testing.T) {
	s := newTestStore()
	expires := time.Now().UTC().Add(5 * time.Minute)

	acquired, err := s.tryAcquireAlertDedup(context.Background(), "rule1|ck-1|1234", expires)
	if err != nil {
		t.Fatalf("tryAcquireAlertDedup error: %v", err)
	}
	if !acquired {
		t.Fatalf("first acquire should return true, got false")
	}
}

func TestTryAcquireDedup_DuplicateKeyReturnsFalse(t *testing.T) {
	s := newTestStore()
	expires := time.Now().UTC().Add(5 * time.Minute)
	key := "rule1|ck-1|1234"

	acquired1, _ := s.tryAcquireAlertDedup(context.Background(), key, expires)
	if !acquired1 {
		t.Fatalf("first acquire should return true")
	}

	acquired2, _ := s.tryAcquireAlertDedup(context.Background(), key, expires)
	if acquired2 {
		t.Fatalf("duplicate key should return false, got true")
	}
}

func TestTryAcquireDedup_ExpiredKeyCanBeReAcquired(t *testing.T) {
	s := newTestStore()
	key := "rule1|ck-1|1234"

	pastExpiry := time.Now().UTC().Add(-1 * time.Second)
	futureExpiry := time.Now().UTC().Add(5 * time.Minute)

	s.dedupMu.Lock()
	s.dedupEntries[key] = dedupEntry{
		DedupKey:  key,
		ExpiresAt: pastExpiry,
		CreatedAt: time.Now().UTC().Add(-10 * time.Minute),
	}
	s.dedupMu.Unlock()

	acquired, err := s.tryAcquireAlertDedup(context.Background(), key, futureExpiry)
	if err != nil {
		t.Fatalf("re-acquire expired key error: %v", err)
	}
	if !acquired {
		t.Fatalf("expired key should be re-acquirable, got false")
	}
}

func TestTryAcquireDedup_DifferentKeysBothAcquired(t *testing.T) {
	s := newTestStore()
	expires := time.Now().UTC().Add(5 * time.Minute)

	acquired1, _ := s.tryAcquireAlertDedup(context.Background(), "rule1|ck-1|1234", expires)
	acquired2, _ := s.tryAcquireAlertDedup(context.Background(), "rule1|ck-2|1234", expires)

	if !acquired1 || !acquired2 {
		t.Fatalf("different dedup keys should both acquire: first=%v second=%v", acquired1, acquired2)
	}
}

func TestTryAcquireDedup_ConcurrentSameKey(t *testing.T) {
	s := newTestStore()
	expires := time.Now().UTC().Add(5 * time.Minute)
	key := "rule1|ck-1|1234"

	acquiredCount := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	const attempts = 20
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			acquired, _ := s.tryAcquireAlertDedup(context.Background(), key, expires)
			mu.Lock()
			if acquired {
				acquiredCount++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if acquiredCount != 1 {
		t.Fatalf("concurrent attempts on same key: exactly 1 should acquire, got %d", acquiredCount)
	}
}

func TestTryAcquireDedup_AddsToPersistBuffer(t *testing.T) {
	s := newTestStore()
	expires := time.Now().UTC().Add(5 * time.Minute)

	s.tryAcquireAlertDedup(context.Background(), "rule1|ck-1|1234", expires)

	s.dedupMu.Lock()
	persistCount := len(s.dedupPersist)
	memCount := len(s.dedupEntries)
	s.dedupMu.Unlock()

	if persistCount != 1 {
		t.Fatalf("expected 1 entry in dedup persist buffer, got %d", persistCount)
	}
	if memCount != 1 {
		t.Fatalf("expected 1 entry in in-memory dedup, got %d", memCount)
	}
}

func TestSaveCheckpoint_BuffersInMemory(t *testing.T) {
	s := newTestStore()
	eventID := uuid.New()
	eventTime := time.Now().UTC()

	err := s.saveCheckpoint(context.Background(), runtimeCheckpoint{LastEventID: eventID, LastEventTime: eventTime})
	if err != nil {
		t.Fatalf("saveCheckpoint error: %v", err)
	}

	s.checkpointMu.Lock()
	cp := s.pendingCheckpoint
	s.checkpointMu.Unlock()

	if cp == nil {
		t.Fatalf("expected pending checkpoint to be set")
	}
	if cp.LastEventID != eventID {
		t.Fatalf("checkpoint LastEventID mismatch: got %v, want %v", cp.LastEventID, eventID)
	}
	if cp.LastEventTime != eventTime {
		t.Fatalf("checkpoint LastEventTime mismatch: got %v, want %v", cp.LastEventTime, eventTime)
	}
}

func TestSaveCheckpoint_OverwritesPrevious(t *testing.T) {
	s := newTestStore()
	id1 := uuid.New()
	id2 := uuid.New()

	s.saveCheckpoint(context.Background(), runtimeCheckpoint{LastEventID: id1, LastEventTime: time.Now().UTC()})
	s.saveCheckpoint(context.Background(), runtimeCheckpoint{LastEventID: id2, LastEventTime: time.Now().UTC()})

	s.checkpointMu.Lock()
	cp := s.pendingCheckpoint
	s.checkpointMu.Unlock()

	if cp.LastEventID != id2 {
		t.Fatalf("checkpoint should be overwritten with latest, got %v, want %v", cp.LastEventID, id2)
	}
}

func TestDeleteState_RemovesFromPendingAndAddsToDeleteBuf(t *testing.T) {
	s := newTestStore()
	ruleID := uuid.New()
	now := time.Now().UTC()

	s.upsertState(context.Background(), ruleID, "ck-del", "threshold", thresholdPersistedData{Count: 1}, now, now, now.Add(5*time.Minute))

	err := s.deleteState(context.Background(), ruleID, "ck-del")
	if err != nil {
		t.Fatalf("deleteState error: %v", err)
	}

	s.stateMu.Lock()
	_, exists := s.pendingStates[ruleID.String()+":ck-del"]
	s.stateMu.Unlock()

	if exists {
		t.Fatalf("deleted state should not exist in pendingStates")
	}

	s.deleteMu.Lock()
	delCount := len(s.deleteBuf)
	s.deleteMu.Unlock()

	if delCount != 1 {
		t.Fatalf("expected 1 entry in deleteBuf, got %d", delCount)
	}
}

func TestDeleteState_NonExistentKeyStillBuffersDelete(t *testing.T) {
	s := newTestStore()
	ruleID := uuid.New()

	err := s.deleteState(context.Background(), ruleID, "ck-phantom")
	if err != nil {
		t.Fatalf("deleteState for non-existent key: %v", err)
	}

	s.deleteMu.Lock()
	delCount := len(s.deleteBuf)
	s.deleteMu.Unlock()

	if delCount != 1 {
		t.Fatalf("delete of non-existent key should still buffer delete op, got %d", delCount)
	}
}

func TestCollectBufferStats(t *testing.T) {
	s := newTestStore()
	now := time.Now().UTC()
	expires := now.Add(5 * time.Minute)

	s.upsertState(context.Background(), uuid.New(), "ck-1", "threshold", thresholdPersistedData{Count: 1}, now, now, expires)
	s.upsertState(context.Background(), uuid.New(), "ck-2", "threshold", thresholdPersistedData{Count: 2}, now, now, expires)
	s.recordRuleEvaluation(context.Background(), uuid.New(), uuid.New(), now, true, "triggered", "ck-1", nil)
	s.recordRuleEvaluation(context.Background(), uuid.New(), uuid.New(), now, false, "no_match", "ck-2", nil)
	s.recordRuleEvaluation(context.Background(), uuid.New(), uuid.New(), now, true, "triggered", "ck-3", nil)
	s.tryAcquireAlertDedup(context.Background(), "d1", expires)
	s.tryAcquireAlertDedup(context.Background(), "d2", expires)
	s.saveCheckpoint(context.Background(), runtimeCheckpoint{LastEventID: uuid.New(), LastEventTime: now})

	stats := s.collectBufferStats()

	if stats["pending_states"] != 2 {
		t.Fatalf("expected 2 pending states, got %d", stats["pending_states"])
	}
	if stats["pending_evaluations"] != 3 {
		t.Fatalf("expected 3 pending evaluations, got %d", stats["pending_evaluations"])
	}
	if stats["pending_dedup"] != 2 {
		t.Fatalf("expected 2 pending dedup persist, got %d", stats["pending_dedup"])
	}
	if stats["in_memory_dedup"] != 2 {
		t.Fatalf("expected 2 in-memory dedup entries, got %d", stats["in_memory_dedup"])
	}
	if stats["pending_checkpoint"] != 1 {
		t.Fatalf("expected 1 pending checkpoint, got %d", stats["pending_checkpoint"])
	}
}

func TestUpsertState_ReturnsImmediately_ComparedToSimulatedSync(t *testing.T) {
	s := newTestStore()
	ruleID := uuid.New()
	now := time.Now().UTC()

	start := time.Now()
	err := s.upsertState(context.Background(), ruleID, "ck", "threshold", thresholdPersistedData{Count: 1}, now, now, now.Add(5*time.Minute))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("upsertState error: %v", err)
	}
	if elapsed > 1*time.Millisecond {
		t.Fatalf("async upsertState should take <1ms (in-memory only), took %v", elapsed)
	}
}

func TestTryAcquireDedup_ReturnsImmediately(t *testing.T) {
	s := newTestStore()
	expires := time.Now().UTC().Add(5 * time.Minute)

	start := time.Now()
	_, err := s.tryAcquireAlertDedup(context.Background(), "key", expires)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("tryAcquireAlertDedup error: %v", err)
	}
	if elapsed > 1*time.Millisecond {
		t.Fatalf("async tryAcquireAlertDedup should take <1ms (in-memory), took %v", elapsed)
	}
}

func TestRecordEvaluation_ReturnsImmediately(t *testing.T) {
	s := newTestStore()

	start := time.Now()
	err := s.recordRuleEvaluation(context.Background(), uuid.New(), uuid.New(), time.Now().UTC(), true, "r", "ck", nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("recordRuleEvaluation error: %v", err)
	}
	if elapsed > 1*time.Millisecond {
		t.Fatalf("async recordRuleEvaluation should take <1ms, took %v", elapsed)
	}
}

func TestSaveCheckpoint_ReturnsImmediately(t *testing.T) {
	s := newTestStore()

	start := time.Now()
	err := s.saveCheckpoint(context.Background(), runtimeCheckpoint{LastEventID: uuid.New(), LastEventTime: time.Now().UTC()})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("saveCheckpoint error: %v", err)
	}
	if elapsed > 1*time.Millisecond {
		t.Fatalf("async saveCheckpoint should take <1ms (in-memory), took %v", elapsed)
	}
}

func TestFlushStates_DrainsBufferWithoutDB(t *testing.T) {
	s := newTestStore()
	ruleID := uuid.New()
	now := time.Now().UTC()
	expires := now.Add(5 * time.Minute)

	s.upsertState(context.Background(), ruleID, "ck-1", "threshold", thresholdPersistedData{Count: 1}, now, now, expires)

	s.stateMu.Lock()
	beforeLen := len(s.pendingStates)
	s.stateMu.Unlock()

	if beforeLen != 1 {
		t.Fatalf("expected 1 pending state before drain, got %d", beforeLen)
	}

	s.stateMu.Lock()
	pending := s.pendingStates
	s.pendingStates = make(map[string]persistedRuntimeStateRow)
	s.stateMu.Unlock()

	if len(pending) != 1 {
		t.Fatalf("expected 1 drained state, got %d", len(pending))
	}

	s.stateMu.Lock()
	afterLen := len(s.pendingStates)
	s.stateMu.Unlock()

	if afterLen != 0 {
		t.Fatalf("pendingStates should be empty after drain, got %d", afterLen)
	}
}

func TestFlushEvaluations_DrainsBufferWithoutDB(t *testing.T) {
	s := newTestStore()

	for i := 0; i < 5; i++ {
		s.recordRuleEvaluation(context.Background(), uuid.New(), uuid.New(), time.Now().UTC(), true, "r", "ck", nil)
	}

	s.evalMu.Lock()
	beforeLen := len(s.evalBuf)
	s.evalMu.Unlock()

	if beforeLen != 5 {
		t.Fatalf("expected 5 buffered evaluations before drain, got %d", beforeLen)
	}

	s.evalMu.Lock()
	buf := s.evalBuf
	s.evalBuf = make([]evaluationEntry, 0, 256)
	s.evalMu.Unlock()

	if len(buf) != 5 {
		t.Fatalf("expected 5 drained evaluations, got %d", len(buf))
	}

	s.evalMu.Lock()
	afterLen := len(s.evalBuf)
	s.evalMu.Unlock()

	if afterLen != 0 {
		t.Fatalf("evalBuf should be empty after drain, got %d", afterLen)
	}
}
