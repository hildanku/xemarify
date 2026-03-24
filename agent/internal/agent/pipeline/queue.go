package pipeline

import (
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
