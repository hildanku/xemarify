package engine

import (
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const defaultStateShardCount = 64
const defaultMaxStatesPerRule = 100000

type State struct {
	Count         int
	FirstSeen     time.Time
	LastSeen      time.Time
	ExpiresAt     time.Time
	LastAlertTime time.Time
	RuleID        uuid.UUID
	LastEventID   uuid.UUID
}

type stateShard struct {
	mu   sync.Mutex
	data map[string]*State
}

// StateStore is a low-contention in-memory store for correlation counters.
// It uses map sharding and periodic TTL cleanup.
type StateStore struct {
	shards           []stateShard
	cleanupInterval  time.Duration
	maxStatesPerRule int
	ruleCounts       map[uuid.UUID]int
	ruleCountsMu     sync.Mutex
	totalEntries     atomic.Int64
	metrics          *EngineMetrics
	log              *logrus.Logger
	stopCh           chan struct{}
	wg               sync.WaitGroup
}

func NewStateStore(cleanupInterval time.Duration, maxStatesPerRule int, log *logrus.Logger, metrics *EngineMetrics) *StateStore {
	if cleanupInterval <= 0 {
		cleanupInterval = 30 * time.Second
	}
	if maxStatesPerRule <= 0 {
		maxStatesPerRule = defaultMaxStatesPerRule
	}

	shards := make([]stateShard, defaultStateShardCount)
	for idx := range shards {
		shards[idx] = stateShard{data: make(map[string]*State)}
	}

	store := &StateStore{
		shards:           shards,
		cleanupInterval:  cleanupInterval,
		maxStatesPerRule: maxStatesPerRule,
		ruleCounts:       make(map[uuid.UUID]int),
		metrics:          metrics,
		log:              log,
		stopCh:           make(chan struct{}),
	}

	store.wg.Add(1)
	go store.cleanupLoop()

	return store
}

func (s *StateStore) Stop() {
	if s == nil {
		return
	}
	close(s.stopCh)
	s.wg.Wait()
}

// Update increments or resets state using an approximate fixed window:
// if now-first_seen > window then reset counter.
func (s *StateStore) Update(rule CompiledRule, key string, eventTime time.Time, eventID uuid.UUID) (State, bool) {
	if eventTime.IsZero() {
		eventTime = time.Now().UTC()
	}

	shard := s.shardForKey(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	state, found := shard.data[key]
	if !found {
		if !s.tryAddRuleState(rule.ID) {
			if s.log != nil {
				s.log.WithFields(logrus.Fields{
					"rule_id":             rule.ID,
					"max_states_per_rule": s.maxStatesPerRule,
				}).Warn("state limit reached for rule")
			}
			return State{}, false
		}

		fresh := &State{
			Count:       1,
			FirstSeen:   eventTime,
			LastSeen:    eventTime,
			ExpiresAt:   eventTime.Add(rule.Window),
			RuleID:      rule.ID,
			LastEventID: eventID,
		}
		shard.data[key] = fresh
		s.updateStateEntriesMetric()
		return *fresh, true
	}

	if eventTime.Sub(state.FirstSeen) > rule.Window {
		state.Count = 1
		state.FirstSeen = eventTime
		state.LastSeen = eventTime
		state.ExpiresAt = eventTime.Add(rule.Window)
		state.LastAlertTime = time.Time{}
		state.LastEventID = eventID
		return *state, true
	}

	state.Count++
	state.LastSeen = eventTime
	state.ExpiresAt = state.FirstSeen.Add(rule.Window)
	state.LastEventID = eventID

	return *state, true
}

func (s *StateStore) MarkAlert(key string, alertTime time.Time) {
	if alertTime.IsZero() {
		alertTime = time.Now().UTC()
	}

	shard := s.shardForKey(key)
	shard.mu.Lock()
	if state, found := shard.data[key]; found {
		state.LastAlertTime = alertTime
	}
	shard.mu.Unlock()
}

func (s *StateStore) Reset(key string) {
	shard := s.shardForKey(key)
	shard.mu.Lock()
	if state, found := shard.data[key]; found {
		delete(shard.data, key)
		s.removeRuleState(state.RuleID)
		s.updateStateEntriesMetric()
	}
	shard.mu.Unlock()
}

func (s *StateStore) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupExpired(time.Now().UTC())
		case <-s.stopCh:
			return
		}
	}
}

func (s *StateStore) cleanupExpired(now time.Time) {
	for idx := range s.shards {
		shard := &s.shards[idx]
		shard.mu.Lock()
		for key, state := range shard.data {
			if now.After(state.ExpiresAt) {
				delete(shard.data, key)
				s.removeRuleState(state.RuleID)
			}
		}
		shard.mu.Unlock()
	}
	s.updateStateEntriesMetric()
}

func (s *StateStore) shardForKey(key string) *stateShard {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(key))
	idx := hasher.Sum32() % uint32(len(s.shards))
	return &s.shards[idx]
}

func (s *StateStore) tryAddRuleState(ruleID uuid.UUID) bool {
	s.ruleCountsMu.Lock()
	defer s.ruleCountsMu.Unlock()

	count := s.ruleCounts[ruleID]
	if count >= s.maxStatesPerRule {
		return false
	}

	s.ruleCounts[ruleID] = count + 1
	s.totalEntries.Add(1)
	return true
}

func (s *StateStore) removeRuleState(ruleID uuid.UUID) {
	s.ruleCountsMu.Lock()
	defer s.ruleCountsMu.Unlock()

	count := s.ruleCounts[ruleID]
	if count <= 1 {
		delete(s.ruleCounts, ruleID)
	} else {
		s.ruleCounts[ruleID] = count - 1
	}
	s.totalEntries.Add(-1)
}

func (s *StateStore) updateStateEntriesMetric() {
	if s.metrics != nil {
		s.metrics.StateEntries.Set(float64(s.totalEntries.Load()))
	}
}
