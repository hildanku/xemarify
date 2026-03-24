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

func RunSender(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	agentID string,
	sessionKey string,
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

			if err := apiclient.PostJSON(ctx, client, apiclient.JoinURL(endpoint, "/api/v1/events"), sessionKey, payload, http.StatusAccepted); err != nil {
				queue.Nack(ids, now, retry)
				log.Printf("event send failed (%d events, queue=%d): %v", len(items), queue.Len(), err)
				return
			}

			queue.Ack(ids)
			eventsDelivered.Add(int64(len(items)))

			if len(items) < batchSize {
				return
			}
		}
	}

	flush()
	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-ticker.C:
			flush()
		}
	}
}

func RunHeartbeat(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	agentID string,
	sessionKey string,
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
		if err := apiclient.PostJSON(ctx, client, apiclient.JoinURL(endpoint, "/api/v1/agents/heartbeat"), sessionKey, payload, http.StatusOK); err != nil {
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
