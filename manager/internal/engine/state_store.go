package engine

import (
	"hash/fnv"
	"sync"
	"time"

	"github.com/google/uuid"
)

const defaultStateShardCount = 64

type State struct {
	Count       int
	FirstSeen   time.Time
	LastSeen    time.Time
	ExpiresAt   time.Time
	LastEventID uuid.UUID
}

type stateShard struct {
	mu   sync.Mutex
	data map[string]*State
}

// StateStore is a low-contention in-memory store for correlation counters.
// It uses map sharding and periodic TTL cleanup.
type StateStore struct {
	shards          []stateShard
	cleanupInterval time.Duration
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

func NewStateStore(cleanupInterval time.Duration) *StateStore {
	if cleanupInterval <= 0 {
		cleanupInterval = 30 * time.Second
	}

	shards := make([]stateShard, defaultStateShardCount)
	for idx := range shards {
		shards[idx] = stateShard{data: make(map[string]*State)}
	}

	store := &StateStore{
		shards:          shards,
		cleanupInterval: cleanupInterval,
		stopCh:          make(chan struct{}),
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
func (s *StateStore) Update(rule CompiledRule, key string, eventTime time.Time, eventID uuid.UUID) State {
	if eventTime.IsZero() {
		eventTime = time.Now().UTC()
	}

	shard := s.shardForKey(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	state, found := shard.data[key]
	if !found || eventTime.Sub(state.FirstSeen) > rule.Window {
		fresh := &State{
			Count:       1,
			FirstSeen:   eventTime,
			LastSeen:    eventTime,
			ExpiresAt:   eventTime.Add(rule.Window),
			LastEventID: eventID,
		}
		shard.data[key] = fresh
		return *fresh
	}

	state.Count++
	state.LastSeen = eventTime
	state.ExpiresAt = state.FirstSeen.Add(rule.Window)
	state.LastEventID = eventID

	return *state
}

func (s *StateStore) Reset(key string) {
	shard := s.shardForKey(key)
	shard.mu.Lock()
	delete(shard.data, key)
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
			}
		}
		shard.mu.Unlock()
	}
}

func (s *StateStore) shardForKey(key string) *stateShard {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(key))
	idx := hasher.Sum32() % uint32(len(s.shards))
	return &s.shards[idx]
}
