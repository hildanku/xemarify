package pipeline

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hildanku/xemarify-agent/internal/agent/model"
)

// RetryPolicy computes wait duration before retrying a failed item.
type RetryPolicy interface {
	NextDelay(attempt int) time.Duration
}

// ExponentialBackoffPolicy doubles delay each attempt, capped at MaxDelay.
type ExponentialBackoffPolicy struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

// NextDelay returns the backoff duration for a given attempt number.
func (p ExponentialBackoffPolicy) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}

	delay := p.BaseDelay
	for i := 1; i < attempt; i++ {
		if delay >= p.MaxDelay {
			return p.MaxDelay
		}
		delay *= 2
	}

	if delay > p.MaxDelay {
		return p.MaxDelay
	}

	return delay
}

// QueueItem is the public view of a dequeued event, exposed to the sender.
type QueueItem struct {
	ID      uint64
	Event   model.IngestEvent
	Attempt int
}

// EventQueue is the interface both MemoryQueue and DiskBackedQueue satisfy.
type EventQueue interface {
	Enqueue(event model.IngestEvent)
	// DequeueReadyBatch moves up to limit items whose readyAt <= now into in-flight state.
	DequeueReadyBatch(now time.Time, limit int) []QueueItem
	// Ack removes successfully delivered items from in-flight.
	Ack(ids []uint64)
	// Nack returns failed items to pending with a new readyAt based on retry policy.
	Nack(ids []uint64, now time.Time, retry RetryPolicy)
	Len() int
}

// queuedEvent is the internal queue entry with scheduling metadata.
type queuedEvent struct {
	id      uint64
	event   model.IngestEvent
	attempt int
	readyAt time.Time // earliest time this item can be dequeued
}

// MemoryQueue is a non-persistent in-memory queue. Data is lost on restart.
type MemoryQueue struct {
	mu       sync.Mutex
	nextID   uint64
	pending  []*queuedEvent
	inFlight map[uint64]*queuedEvent // items currently being sent
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		pending:  make([]*queuedEvent, 0, 1024),
		inFlight: make(map[uint64]*queuedEvent),
	}
}

func (q *MemoryQueue) Enqueue(event model.IngestEvent) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.nextID++
	q.pending = append(q.pending, &queuedEvent{
		id:      q.nextID,
		event:   event,
		attempt: 0,
		readyAt: time.Now().UTC(),
	})
}

// DequeueReadyBatch picks items ready by now, moves them to in-flight, returns up to limit.
func (q *MemoryQueue) DequeueReadyBatch(now time.Time, limit int) []QueueItem {
	if limit <= 0 {
		return nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	items := make([]QueueItem, 0, limit)
	remaining := make([]*queuedEvent, 0, len(q.pending))

	for _, item := range q.pending {
		if len(items) >= limit {
			remaining = append(remaining, item)
			continue
		}

		// Skip items not yet ready (backoff delay not elapsed).
		if item.readyAt.After(now) {
			remaining = append(remaining, item)
			continue
		}

		q.inFlight[item.id] = item
		items = append(items, QueueItem{
			ID:      item.id,
			Event:   item.event,
			Attempt: item.attempt,
		})
	}

	q.pending = remaining
	return items
}

// Ack drops successfully delivered items from in-flight.
func (q *MemoryQueue) Ack(ids []uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, id := range ids {
		delete(q.inFlight, id)
	}
}

// Nack returns failed items to pending with incremented attempt and new readyAt.
func (q *MemoryQueue) Nack(ids []uint64, now time.Time, retry RetryPolicy) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, id := range ids {
		item, ok := q.inFlight[id]
		if !ok {
			continue
		}

		item.attempt++
		item.readyAt = now.Add(retry.NextDelay(item.attempt))
		q.pending = append(q.pending, item)
		delete(q.inFlight, id)
	}
}

// Len returns total count of pending + in-flight items.
func (q *MemoryQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.pending) + len(q.inFlight)
}

// queueSnapshot is the JSON structure persisted to disk.
type queueSnapshot struct {
	NextID uint64              `json:"next_id"`
	Items  []queueSnapshotItem `json:"items"`
}

// queueSnapshotItem serializes a single queued event including its in-flight state.
type queueSnapshotItem struct {
	ID       uint64            `json:"id"`
	Event    model.IngestEvent `json:"event"`
	Attempt  int               `json:"attempt"`
	ReadyAt  time.Time         `json:"ready_at"`
	InFlight bool              `json:"in_flight"` // if true, readyAt is reset to now on load
}

// DiskBackedQueue persists the queue to a JSON file after every mutation.
// Enqueue blocks when usedSize would exceed maxBytes until space is freed by Ack.
type DiskBackedQueue struct {
	mu       sync.Mutex
	cond     *sync.Cond // signals waiters in Enqueue when space is freed
	path     string
	maxBytes int64
	nextID   uint64
	pending  []*queuedEvent
	inFlight map[uint64]*queuedEvent
	usedSize int64 // total JSON-encoded size of all tracked events
}

// NewDiskBackedQueue creates a queue backed by path, loading any existing snapshot.
func NewDiskBackedQueue(path string, maxBytes int64) (*DiskBackedQueue, error) {
	if path == "" {
		return nil, errors.New("disk buffer path is required")
	}

	if maxBytes <= 0 {
		return nil, errors.New("disk buffer max bytes must be > 0")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}

	q := &DiskBackedQueue{
		path:     path,
		maxBytes: maxBytes,
		pending:  make([]*queuedEvent, 0, 1024),
		inFlight: make(map[uint64]*queuedEvent),
	}
	q.cond = sync.NewCond(&q.mu)

	if err := q.load(); err != nil {
		return nil, err
	}

	// Write initial snapshot to verify the file is writable.
	if err := q.persistLocked(); err != nil {
		return nil, err
	}

	return q, nil
}

// Enqueue adds an event to the queue. Blocks if the buffer is full until space is freed.
// Drops events that exceed maxBytes on their own.
func (q *DiskBackedQueue) Enqueue(event model.IngestEvent) {
	eventSize := encodedEventSize(event)
	if eventSize > q.maxBytes {
		log.Printf("drop oversized event: event_size=%d max_buffer_size=%d", eventSize, q.maxBytes)
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Wait until there's enough space for this event.
	for q.usedSize+eventSize > q.maxBytes {
		q.cond.Wait()
	}

	q.nextID++
	q.pending = append(q.pending, &queuedEvent{
		id:      q.nextID,
		event:   event,
		attempt: 0,
		readyAt: time.Now().UTC(),
	})
	q.usedSize += eventSize

	if err := q.persistLocked(); err != nil {
		log.Printf("failed to persist queue enqueue: %v", err)
	}
}

// DequeueReadyBatch picks items ready by now, moves them to in-flight, persists state.
func (q *DiskBackedQueue) DequeueReadyBatch(now time.Time, limit int) []QueueItem {
	if limit <= 0 {
		return nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	items := make([]QueueItem, 0, limit)
	remaining := make([]*queuedEvent, 0, len(q.pending))

	for _, item := range q.pending {
		if len(items) >= limit {
			remaining = append(remaining, item)
			continue
		}

		// Skip items still under backoff.
		if item.readyAt.After(now) {
			remaining = append(remaining, item)
			continue
		}

		q.inFlight[item.id] = item
		items = append(items, QueueItem{
			ID:      item.id,
			Event:   item.event,
			Attempt: item.attempt,
		})
	}

	q.pending = remaining

	if len(items) > 0 {
		if err := q.persistLocked(); err != nil {
			log.Printf("failed to persist queue dequeue: %v", err)
		}
	}

	return items
}

// Ack removes delivered items, frees their byte budget, wakes blocked Enqueue callers.
func (q *DiskBackedQueue) Ack(ids []uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	released := int64(0)
	for _, id := range ids {
		item, ok := q.inFlight[id]
		if !ok {
			continue
		}

		released += encodedEventSize(item.event)
		delete(q.inFlight, id)
	}

	q.usedSize -= released
	if q.usedSize < 0 {
		q.usedSize = 0
	}

	if len(ids) > 0 {
		if err := q.persistLocked(); err != nil {
			log.Printf("failed to persist queue ack: %v", err)
		}
	}

	// Wake any Enqueue callers waiting for space.
	if released > 0 {
		q.cond.Broadcast()
	}
}

// Nack returns failed items to pending with incremented attempt and backoff readyAt.
func (q *DiskBackedQueue) Nack(ids []uint64, now time.Time, retry RetryPolicy) {
	q.mu.Lock()
	defer q.mu.Unlock()

	changed := false
	for _, id := range ids {
		item, ok := q.inFlight[id]
		if !ok {
			continue
		}

		item.attempt++
		item.readyAt = now.Add(retry.NextDelay(item.attempt))
		q.pending = append(q.pending, item)
		delete(q.inFlight, id)
		changed = true
	}

	if changed {
		if err := q.persistLocked(); err != nil {
			log.Printf("failed to persist queue nack: %v", err)
		}
	}
}

// Len returns total count of pending + in-flight items.
func (q *DiskBackedQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.pending) + len(q.inFlight)
}

// load reads and deserializes the snapshot file into queue state.
// In-flight items from a prior run are reset to readyAt=now (immediate retry).
func (q *DiskBackedQueue) load() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	data, err := os.ReadFile(q.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Fresh start; initialize empty state.
			q.nextID = 0
			q.pending = make([]*queuedEvent, 0, 1024)
			q.inFlight = make(map[uint64]*queuedEvent)
			q.usedSize = 0
			return nil
		}
		return err
	}

	if len(data) == 0 {
		q.nextID = 0
		q.pending = make([]*queuedEvent, 0, 1024)
		q.inFlight = make(map[uint64]*queuedEvent)
		q.usedSize = 0
		return nil
	}

	var snapshot queueSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}

	q.nextID = snapshot.NextID
	q.pending = make([]*queuedEvent, 0, len(snapshot.Items))
	q.inFlight = make(map[uint64]*queuedEvent)
	q.usedSize = 0

	now := time.Now().UTC()
	for _, item := range snapshot.Items {
		queued := &queuedEvent{
			id:      item.ID,
			event:   item.Event,
			attempt: item.Attempt,
			readyAt: item.ReadyAt,
		}

		// Items that were in-flight when the process died are retried immediately.
		if item.InFlight {
			queued.readyAt = now
		}

		q.pending = append(q.pending, queued)
		q.usedSize += encodedEventSize(queued.event)
	}

	return nil
}

// persistLocked serializes full queue state to disk atomically.
// Must be called with q.mu held.
func (q *DiskBackedQueue) persistLocked() error {
	snapshot := queueSnapshot{
		NextID: q.nextID,
		Items:  make([]queueSnapshotItem, 0, len(q.pending)+len(q.inFlight)),
	}

	for _, item := range q.pending {
		snapshot.Items = append(snapshot.Items, queueSnapshotItem{
			ID:       item.id,
			Event:    item.event,
			Attempt:  item.attempt,
			ReadyAt:  item.readyAt,
			InFlight: false,
		})
	}

	for _, item := range q.inFlight {
		snapshot.Items = append(snapshot.Items, queueSnapshotItem{
			ID:       item.id,
			Event:    item.event,
			Attempt:  item.attempt,
			ReadyAt:  item.readyAt,
			InFlight: true,
		})
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	return writeAtomicSynced(q.path, data)
}

// writeAtomicSynced writes data to path via a temp file + rename to avoid partial writes.
// Both the file and its parent directory are fsynced for crash safety.
func writeAtomicSynced(path string, data []byte) error {
	tmpPath := path + ".tmp"

	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	// Atomic swap: rename is guaranteed to be atomic on POSIX.
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	// Fsync the directory so the rename is durable.
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()

	return dir.Sync()
}

// encodedEventSize returns JSON byte size of event, used for buffer accounting.
func encodedEventSize(event model.IngestEvent) int64 {
	b, err := json.Marshal(event)
	if err != nil {
		return 0
	}

	return int64(len(b))
}
