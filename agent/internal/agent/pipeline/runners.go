package pipeline

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/hildanku/xemarify-agent/internal/agent/apiclient"
	"github.com/hildanku/xemarify-agent/internal/agent/model"
)

// RunIngestor drains the input channel and pushes each event into the queue.
// Stops when ctx is cancelled.
func RunIngestor(ctx context.Context, input <-chan model.IngestEvent, queue EventQueue) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-input:
			queue.Enqueue(event)
		}
	}
}

// RunSender flushes ready batches from the queue to the events endpoint on a ticker.
// On send failure, items are nack'd back with retry backoff.
// On ctx cancel, one final flush is attempted before exit.
func RunSender(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	agentID string,
	agentSecret string,
	queue EventQueue,
	retry RetryPolicy,
	batchSize int,
	batchFlushEvery time.Duration,
	eventsDelivered *atomic.Int64,
) {
	ticker := time.NewTicker(batchFlushEvery)
	defer ticker.Stop()

	flush := func() {
		now := time.Now().UTC()
		for {
			items := queue.DequeueReadyBatch(now, batchSize)
			if len(items) == 0 {
				return
			}

			// Separate IDs and events for the batch payload.
			ids := make([]uint64, 0, len(items))
			events := make([]model.IngestEvent, 0, len(items))
			for _, item := range items {
				ids = append(ids, item.ID)
				events = append(events, item.Event)
			}

			payload := model.EventBatch{
				AgentID: agentID,
				Events:  events,
			}

			if err := apiclient.PostJSON(ctx, client, apiclient.JoinURL(endpoint, "/api/v1/events"), agentSecret, payload, http.StatusAccepted); err != nil {
				// Send failed; return items to queue with backoff delay.
				queue.Nack(ids, now, retry)
				log.Printf("event send failed (%d events, queue=%d): %v", len(items), queue.Len(), err)
				return
			}

			queue.Ack(ids)
			eventsDelivered.Add(int64(len(items)))

			// Batch smaller than limit means no more ready items.
			if len(items) < batchSize {
				return
			}
		}
	}

	// Flush once immediately on start, then on every tick.
	flush()
	for {
		select {
		case <-ctx.Done():
			flush() // drain remaining items before shutdown
			return
		case <-ticker.C:
			flush()
		}
	}
}

// RunHeartbeat sends periodic heartbeats with uptime and delivered event count.
// Sends once immediately on start, then on every interval tick.
func RunHeartbeat(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	agentID string,
	agentSecret string,
	interval time.Duration,
	eventsDelivered *atomic.Int64,
	startedAt time.Time,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	send := func() {
		payload := model.HeartbeatRequest{
			AgentID:    agentID,
			EventsSent: eventsDelivered.Load(),
			Uptime:     int64(time.Since(startedAt).Seconds()),
		}
		if err := apiclient.PostJSON(ctx, client, apiclient.JoinURL(endpoint, "/api/v1/agents/heartbeat"), agentSecret, payload, http.StatusOK); err != nil {
			log.Printf("heartbeat failed: %v", err)
		}
	}

	send()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			send()
		}
	}
}
