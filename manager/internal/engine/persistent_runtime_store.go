package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

const defaultPersistenceFlushInterval = 5 * time.Second

type runtimeCheckpoint struct {
	LastEventID   uuid.UUID
	LastEventTime time.Time
}

type persistedRuntimeStateRow struct {
	RuleID         uuid.UUID
	CorrelationKey string
	StateType      string
	StateData      []byte
	FirstSeenAt    time.Time
	LastSeenAt     time.Time
	ExpiresAt      time.Time
}

type evaluationEntry struct {
	RuleID         uuid.UUID
	EventID        uuid.UUID
	ReceivedAt     time.Time
	Matched        bool
	Reason         string
	CorrelationKey string
	Details        []byte
}

type dedupEntry struct {
	DedupKey  string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type persistentRuntimeStore struct {
	db             *pgxpool.Pool
	log            *logrus.Logger
	metrics        *EngineMetrics
	flushInterval  time.Duration
	stopCh         chan struct{}
	stopWG         sync.WaitGroup

	stateMu       sync.Mutex
	pendingStates map[string]persistedRuntimeStateRow

	deleteMu    sync.Mutex
	deleteBuf   []deleteEntry

	evalMu    sync.Mutex
	evalBuf   []evaluationEntry

	dedupMu       sync.Mutex
	dedupEntries  map[string]dedupEntry
	dedupPersist  []dedupEntry

	checkpointMu sync.Mutex
	pendingCheckpoint *runtimeCheckpoint
}

type deleteEntry struct {
	RuleID         uuid.UUID
	CorrelationKey string
}

func newPersistentRuntimeStore(db *pgxpool.Pool, log *logrus.Logger, metrics *EngineMetrics) *persistentRuntimeStore {
	return &persistentRuntimeStore{
		db:             db,
		log:            log,
		metrics:        metrics,
		flushInterval:  defaultPersistenceFlushInterval,
		stopCh:         make(chan struct{}),
		pendingStates:  make(map[string]persistedRuntimeStateRow),
		dedupEntries:   make(map[string]dedupEntry),
		evalBuf:        make([]evaluationEntry, 0, 256),
		deleteBuf:      make([]deleteEntry, 0, 16),
		dedupPersist:   make([]dedupEntry, 0, 64),
	}
}

func (s *persistentRuntimeStore) Start() {
	s.stopWG.Add(1)
	go s.periodicFlush()
	s.log.WithField("flush_interval", s.flushInterval).Info("persistent runtime store batch writer started")
}

func (s *persistentRuntimeStore) Stop() {
	close(s.stopCh)
	s.stopWG.Wait()
	s.flushAll(context.Background())
	s.log.Info("persistent runtime store batch writer stopped, final flush complete")
}

func (s *persistentRuntimeStore) periodicFlush() {
	defer s.stopWG.Done()
	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.flushAll(context.Background())
		case <-s.stopCh:
			return
		}
	}
}

func (s *persistentRuntimeStore) flushAll(ctx context.Context) {
	s.flushStates(ctx)
	s.flushEvaluations(ctx)
	s.flushDedup(ctx)
	s.flushCheckpoint(ctx)
	s.flushDeletes(ctx)
}

func (s *persistentRuntimeStore) upsertState(
	ctx context.Context,
	ruleID uuid.UUID,
	correlationKey string,
	stateType string,
	stateData any,
	firstSeenAt, lastSeenAt, expiresAt time.Time,
) error {
	payload, err := json.Marshal(stateData)
	if err != nil {
		return err
	}

	key := ruleID.String() + ":" + correlationKey

	s.stateMu.Lock()
	s.pendingStates[key] = persistedRuntimeStateRow{
		RuleID:         ruleID,
		CorrelationKey: correlationKey,
		StateType:      stateType,
		StateData:      payload,
		FirstSeenAt:    firstSeenAt,
		LastSeenAt:     lastSeenAt,
		ExpiresAt:      expiresAt,
	}
	s.stateMu.Unlock()

	s.metrics.StateBufferDepth.Inc()
	return nil
}

func (s *persistentRuntimeStore) deleteState(ctx context.Context, ruleID uuid.UUID, correlationKey string) error {
	s.deleteMu.Lock()
	s.deleteBuf = append(s.deleteBuf, deleteEntry{RuleID: ruleID, CorrelationKey: correlationKey})
	s.deleteMu.Unlock()

	key := ruleID.String() + ":" + correlationKey
	s.stateMu.Lock()
	delete(s.pendingStates, key)
	s.stateMu.Unlock()

	return nil
}

func (s *persistentRuntimeStore) loadActiveStates(ctx context.Context) ([]persistedRuntimeStateRow, error) {
	const q = `
		SELECT rule_id, correlation_key, state_type, state_data, first_seen_at, last_seen_at, expires_at
		FROM correlation_state
		WHERE expires_at > NOW()
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]persistedRuntimeStateRow, 0, 128)
	for rows.Next() {
		var row persistedRuntimeStateRow
		if err := rows.Scan(
			&row.RuleID,
			&row.CorrelationKey,
			&row.StateType,
			&row.StateData,
			&row.FirstSeenAt,
			&row.LastSeenAt,
			&row.ExpiresAt,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *persistentRuntimeStore) pruneExpiredStates(ctx context.Context) error {
	const q = `DELETE FROM correlation_state WHERE expires_at <= NOW()`
	_, err := s.db.Exec(ctx, q)
	return err
}

func (s *persistentRuntimeStore) tryAcquireAlertDedup(ctx context.Context, dedupKey string, expiresAt time.Time) (bool, error) {
	now := time.Now().UTC()

	s.dedupMu.Lock()
	entry, exists := s.dedupEntries[dedupKey]
	if exists && entry.ExpiresAt.After(now) {
		s.dedupMu.Unlock()
		return false, nil
	}
	s.dedupEntries[dedupKey] = dedupEntry{
		DedupKey:  dedupKey,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
	s.dedupPersist = append(s.dedupPersist, dedupEntry{
		DedupKey:  dedupKey,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	})
	s.dedupMu.Unlock()

	s.metrics.DedupBufferDepth.Inc()
	return true, nil
}

func (s *persistentRuntimeStore) loadActiveDedupEntries(ctx context.Context) error {
	const q = `
		SELECT dedup_key, expires_at, created_at
		FROM detection_alert_dedup
		WHERE expires_at > NOW()
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()

	s.dedupMu.Lock()
	for rows.Next() {
		var entry dedupEntry
		if err := rows.Scan(&entry.DedupKey, &entry.ExpiresAt, &entry.CreatedAt); err != nil {
			s.dedupMu.Unlock()
			return err
		}
		s.dedupEntries[entry.DedupKey] = entry
	}
	s.dedupMu.Unlock()

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

func (s *persistentRuntimeStore) recordRuleEvaluation(
	ctx context.Context,
	ruleID uuid.UUID,
	eventID uuid.UUID,
	receivedAt time.Time,
	matched bool,
	reason string,
	correlationKey string,
	evaluationDetails map[string]any,
) error {
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}

	if evaluationDetails == nil {
		evaluationDetails = map[string]any{}
	}

	payload, err := json.Marshal(evaluationDetails)
	if err != nil {
		return err
	}

	s.evalMu.Lock()
	s.evalBuf = append(s.evalBuf, evaluationEntry{
		RuleID:         ruleID,
		EventID:        eventID,
		ReceivedAt:     receivedAt,
		Matched:        matched,
		Reason:         reason,
		CorrelationKey: correlationKey,
		Details:        payload,
	})
	s.evalMu.Unlock()

	s.metrics.EvalBufferDepth.Inc()
	return nil
}

func (s *persistentRuntimeStore) saveCheckpoint(ctx context.Context, checkpoint runtimeCheckpoint) error {
	s.checkpointMu.Lock()
	s.pendingCheckpoint = &checkpoint
	s.checkpointMu.Unlock()
	return nil
}

func (s *persistentRuntimeStore) loadCheckpoint(ctx context.Context) (runtimeCheckpoint, bool, error) {
	const q = `
		SELECT last_event_id, last_event_time
		FROM engine_processing_checkpoint
		WHERE engine_name = 'rule_engine'
	`

	var checkpoint runtimeCheckpoint
	err := s.db.QueryRow(ctx, q).Scan(&checkpoint.LastEventID, &checkpoint.LastEventTime)
	if err != nil {
		if err == pgx.ErrNoRows {
			return runtimeCheckpoint{}, false, nil
		}
		return runtimeCheckpoint{}, false, err
	}

	return checkpoint, true, nil
}

func (s *persistentRuntimeStore) flushStates(ctx context.Context) {
	s.stateMu.Lock()
	pending := s.pendingStates
	s.pendingStates = make(map[string]persistedRuntimeStateRow)
	s.stateMu.Unlock()

	if len(pending) == 0 {
		return
	}

	s.metrics.StateFlushTotal.Inc()
	s.metrics.StateFlushBatchSize.Observe(float64(len(pending)))
	s.metrics.StateBufferDepth.Sub(float64(len(pending)))

	batch := &pgx.Batch{}
	const q = `
		INSERT INTO correlation_state (
			id,
			rule_id,
			correlation_key,
			state_type,
			state_data,
			first_seen_at,
			last_seen_at,
			expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (rule_id, correlation_key)
		DO UPDATE SET
			state_type = EXCLUDED.state_type,
			state_data = EXCLUDED.state_data,
			first_seen_at = EXCLUDED.first_seen_at,
			last_seen_at = EXCLUDED.last_seen_at,
			expires_at = EXCLUDED.expires_at
	`

	for _, row := range pending {
		batch.Queue(q,
			uuid.New(),
			row.RuleID,
			row.CorrelationKey,
			row.StateType,
			row.StateData,
			row.FirstSeenAt,
			row.LastSeenAt,
			row.ExpiresAt,
		)
	}

	br := s.db.SendBatch(ctx, batch)
	defer br.Close()

	failed := 0
	for range pending {
		if _, err := br.Exec(); err != nil {
			failed++
			s.log.WithError(err).Warn("state batch upsert failed for individual row")
		}
	}

	if failed > 0 {
		s.metrics.StateFlushFailed.Inc()
		s.log.WithField("failed_count", failed).Warn("some state batch upserts failed")
	}
}

func (s *persistentRuntimeStore) flushEvaluations(ctx context.Context) {
	s.evalMu.Lock()
	buf := s.evalBuf
	s.evalBuf = make([]evaluationEntry, 0, 256)
	s.evalMu.Unlock()

	if len(buf) == 0 {
		return
	}

	s.metrics.EvalFlushTotal.Inc()
	s.metrics.EvalFlushBatchSize.Observe(float64(len(buf)))
	s.metrics.EvalBufferDepth.Sub(float64(len(buf)))

	batch := &pgx.Batch{}
	const q = `
		INSERT INTO rule_evaluations (
			rule_id,
			event_id,
			received_at,
			matched,
			reason,
			correlation_key,
			evaluation_details,
			evaluated_at,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`

	for _, entry := range buf {
		batch.Queue(q,
			entry.RuleID,
			entry.EventID,
			entry.ReceivedAt,
			entry.Matched,
			entry.Reason,
			entry.CorrelationKey,
			entry.Details,
		)
	}

	br := s.db.SendBatch(ctx, batch)
	defer br.Close()

	failed := 0
	for range buf {
		if _, err := br.Exec(); err != nil {
			failed++
			s.log.WithError(err).Warn("evaluation batch insert failed for individual row")
		}
	}

	if failed > 0 {
		s.metrics.EvalFlushFailed.Inc()
		s.log.WithField("failed_count", failed).Warn("some evaluation batch inserts failed")
	}
}

func (s *persistentRuntimeStore) flushDedup(ctx context.Context) {
	s.dedupMu.Lock()
	buf := s.dedupPersist
	s.dedupPersist = make([]dedupEntry, 0, 64)
	s.dedupMu.Unlock()

	now := time.Now().UTC()

	s.dedupMu.Lock()
	staleKeys := make([]string, 0)
	for key, entry := range s.dedupEntries {
		if entry.ExpiresAt.Before(now) || entry.ExpiresAt.Equal(now) {
			staleKeys = append(staleKeys, key)
		}
	}
	for _, key := range staleKeys {
		delete(s.dedupEntries, key)
	}
	s.dedupMu.Unlock()

	if len(buf) == 0 && len(staleKeys) == 0 {
		return
	}

	s.metrics.DedupFlushTotal.Inc()
	batchOps := 0

	batch := &pgx.Batch{}

	if len(staleKeys) > 0 {
		for _, key := range staleKeys {
			batch.Queue(`DELETE FROM detection_alert_dedup WHERE dedup_key = $1`, key)
			batchOps++
		}
	}

	if len(buf) > 0 {
		s.metrics.DedupFlushBatchSize.Observe(float64(len(buf)))
		s.metrics.DedupBufferDepth.Sub(float64(len(buf)))

		const insertDedup = `
			INSERT INTO detection_alert_dedup (dedup_key, expires_at, created_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (dedup_key) DO NOTHING
		`
		for _, entry := range buf {
			batch.Queue(insertDedup, entry.DedupKey, entry.ExpiresAt, entry.CreatedAt)
			batchOps++
		}
	}

	if batchOps == 0 {
		return
	}

	br := s.db.SendBatch(ctx, batch)
	defer br.Close()

	failed := 0
	for i := 0; i < batchOps; i++ {
		if _, err := br.Exec(); err != nil {
			failed++
			s.log.WithError(err).Warn("dedup batch operation failed")
		}
	}

	if failed > 0 {
		s.metrics.DedupFlushFailed.Inc()
		s.log.WithField("failed_count", failed).Warn("some dedup batch operations failed")
	}
}

func (s *persistentRuntimeStore) flushCheckpoint(ctx context.Context) {
	s.checkpointMu.Lock()
	cp := s.pendingCheckpoint
	s.pendingCheckpoint = nil
	s.checkpointMu.Unlock()

	if cp == nil {
		return
	}

	const q = `
		INSERT INTO engine_processing_checkpoint (engine_name, last_event_id, last_event_time, updated_at)
		VALUES ('rule_engine', $1, $2, NOW())
		ON CONFLICT (engine_name)
		DO UPDATE SET
			last_event_id = EXCLUDED.last_event_id,
			last_event_time = EXCLUDED.last_event_time,
			updated_at = NOW()
	`

	if _, err := s.db.Exec(ctx, q, cp.LastEventID, cp.LastEventTime); err != nil {
		s.log.WithError(err).Warn("checkpoint upsert failed")
	}
}

func (s *persistentRuntimeStore) flushDeletes(ctx context.Context) {
	s.deleteMu.Lock()
	buf := s.deleteBuf
	s.deleteBuf = make([]deleteEntry, 0, 16)
	s.deleteMu.Unlock()

	if len(buf) == 0 {
		return
	}

	batch := &pgx.Batch{}
	const q = `
		DELETE FROM correlation_state
		WHERE rule_id = $1
		  AND correlation_key = $2
	`

	for _, entry := range buf {
		batch.Queue(q, entry.RuleID, entry.CorrelationKey)
	}

	br := s.db.SendBatch(ctx, batch)
	defer br.Close()

	for range buf {
		if _, err := br.Exec(); err != nil {
			s.log.WithError(err).Warn("state batch delete failed for individual row")
		}
	}
}

func (s *persistentRuntimeStore) collectBufferStats() map[string]int {
	s.stateMu.Lock()
	stateCount := len(s.pendingStates)
	s.stateMu.Unlock()

	s.evalMu.Lock()
	evalCount := len(s.evalBuf)
	s.evalMu.Unlock()

	s.dedupMu.Lock()
	dedupCount := len(s.dedupPersist)
	dedupMemCount := len(s.dedupEntries)
	s.dedupMu.Unlock()

	s.checkpointMu.Lock()
	hasCheckpoint := s.pendingCheckpoint != nil
	s.checkpointMu.Unlock()

	cpVal := 0
	if hasCheckpoint {
		cpVal = 1
	}

	return map[string]int{
		"pending_states":      stateCount,
		"pending_evaluations": evalCount,
		"pending_dedup":       dedupCount,
		"in_memory_dedup":     dedupMemCount,
		"pending_checkpoint":  cpVal,
	}
}

func (s *persistentRuntimeStore) bufferSummary() string {
	stats := s.collectBufferStats()
	return fmt.Sprintf("states=%d evals=%d dedup=%d mem_dedup=%d cp=%v",
		stats["pending_states"],
		stats["pending_evaluations"],
		stats["pending_dedup"],
		stats["in_memory_dedup"],
		stats["pending_checkpoint"] > 0 || s.pendingCheckpoint != nil,
	)
}
