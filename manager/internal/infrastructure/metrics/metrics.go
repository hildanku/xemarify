package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus instrumentation for the ingestion pipeline.
type Metrics struct {
	EventsReceived   *prometheus.CounterVec
	EventsFailed     *prometheus.CounterVec
	IngestionLatency prometheus.Histogram
	DBInsertLatency  prometheus.Histogram
}

// New registers and returns all application metrics.
// Safe to call once at startup — promauto panics on duplicate registration.
func New() *Metrics {
	return &Metrics{
		EventsReceived: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "xemarify_events_received_total",
			Help: "Total number of events received from agents.",
		}, []string{"agent_id"}),

		EventsFailed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "xemarify_events_failed_total",
			Help: "Total number of events that failed to be ingested.",
		}, []string{"reason"}),

		IngestionLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "xemarify_ingestion_duration_seconds",
			Help:    "End-to-end latency for the full event ingestion pipeline.",
			Buckets: prometheus.DefBuckets,
		}),

		DBInsertLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "xemarify_db_insert_duration_seconds",
			Help:    "Latency of the database insert operation for events.",
			Buckets: prometheus.DefBuckets,
		}),
	}
}
