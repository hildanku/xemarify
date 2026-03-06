package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// agentLimiter holds a token-bucket rate limiter and the time it was last used.
// The lastSeen field is used by the cleanup goroutine to evict stale entries.
type agentLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiterConfig controls the per-agent token bucket parameters.
type RateLimiterConfig struct {
	// EventsPerSecond is the sustained rate each agent is allowed.
	EventsPerSecond rate.Limit
	// Burst is the maximum number of events an agent may send in a single burst.
	Burst int
}

// DefaultRateLimiterConfig returns a sensible default for a SOC ingestion endpoint.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		EventsPerSecond: 1000,
		Burst:           2000,
	}
}

type rateLimiterStore struct {
	mu       sync.Mutex
	limiters map[uuid.UUID]*agentLimiter
	cfg      RateLimiterConfig
}

func newRateLimiterStore(cfg RateLimiterConfig) *rateLimiterStore {
	s := &rateLimiterStore{
		limiters: make(map[uuid.UUID]*agentLimiter),
		cfg:      cfg,
	}
	// Background cleanup: remove limiters for agents not seen in 10 minutes.
	go s.cleanup(10 * time.Minute)
	return s
}

func (s *rateLimiterStore) get(agentID uuid.UUID) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.limiters[agentID]
	if !ok {
		entry = &agentLimiter{
			limiter: rate.NewLimiter(s.cfg.EventsPerSecond, s.cfg.Burst),
		}
		s.limiters[agentID] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func (s *rateLimiterStore) cleanup(ttl time.Duration) {
	ticker := time.NewTicker(ttl)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		for id, entry := range s.limiters {
			if time.Since(entry.lastSeen) > ttl {
				delete(s.limiters, id)
			}
		}
		s.mu.Unlock()
	}
}

// AgentRateLimit returns a Gin middleware that applies a per-agent token-bucket
// rate limiter. It expects AgentAuth to have run first so the agent is available
// in the context.
func AgentRateLimit(cfg RateLimiterConfig, log *logrus.Logger) gin.HandlerFunc {
	store := newRateLimiterStore(cfg)

	return func(c *gin.Context) {
		agent := AgentFromContext(c)
		if agent == nil {
			// Should not happen when middleware chain is correct; fail open.
			c.Next()
			return
		}

		limiter := store.get(agent.ID)
		if !limiter.Allow() {
			log.WithFields(logrus.Fields{
				"agent_id":   agent.ID,
				"agent_name": agent.Name,
			}).Warn("rate limit exceeded")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
