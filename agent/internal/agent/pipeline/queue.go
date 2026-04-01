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

type RetryPolicy interface {
	NextDelay(attempt int) time.Duration
}

type ExponentialBackoffPolicy struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

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

type QueueItem struct {
	ID      uint64
	Event   model.IngestEvent
	Attempt int
}

type EventQueue interface {
	Enqueue(event model.IngestEvent)
	DequeueReadyBatch(now time.Time, limit int) []QueueItem
	Ack(ids []uint64)
	Nack(ids []uint64, now time.Time, retry RetryPolicy)
	Len() int
}

type queuedEvent struct {
	id      uint64
	event   model.IngestEvent
	attempt int
	readyAt time.Time
}

type MemoryQueue struct {
	mu       sync.Mutex
	nextID   uint64
	pending  []*queuedEvent
	inFlight map[uint64]*queuedEvent
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

func (q *MemoryQueue) Ack(ids []uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, id := range ids {
		delete(q.inFlight, id)
	}
}

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

func (q *MemoryQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.pending) + len(q.inFlight)
}

type queueSnapshot struct {
	NextID uint64              `json:"next_id"`
	Items  []queueSnapshotItem `json:"items"`
}

type queueSnapshotItem struct {
	ID       uint64            `json:"id"`
	Event    model.IngestEvent `json:"event"`
	Attempt  int               `json:"attempt"`
	ReadyAt  time.Time         `json:"ready_at"`
	InFlight bool              `json:"in_flight"`
}

type DiskBackedQueue struct {
	mu       sync.Mutex
	cond     *sync.Cond
	path     string
	maxBytes int64
	nextID   uint64
	pending  []*queuedEvent
	inFlight map[uint64]*queuedEvent
	usedSize int64
}

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

	if err := q.persistLocked(); err != nil {
		return nil, err
	}

	return q, nil
}

func (q *DiskBackedQueue) Enqueue(event model.IngestEvent) {
	eventSize := encodedEventSize(event)
	if eventSize > q.maxBytes {
		log.Printf("drop oversized event: event_size=%d max_buffer_size=%d", eventSize, q.maxBytes)
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

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

	if released > 0 {
		q.cond.Broadcast()
	}
}

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

func (q *DiskBackedQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.pending) + len(q.inFlight)
}

func (q *DiskBackedQueue) load() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	data, err := os.ReadFile(q.path)
	if err != nil {
		if os.IsNotExist(err) {
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

		if item.InFlight {
			queued.readyAt = now
		}

		q.pending = append(q.pending, queued)
		q.usedSize += encodedEventSize(queued.event)
	}

	return nil
}

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

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()

	return dir.Sync()
}

func encodedEventSize(event model.IngestEvent) int64 {
	b, err := json.Marshal(event)
	if err != nil {
		return 0
	}

	return int64(len(b))
}
