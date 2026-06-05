package middleware

import (
	"sync"
	"time"

	"github.com/hildanku/xemarify/internal/modules/agent/domain"
)

const (
	// agentAuthCacheTTL is how long a successfully validated agent record
	// is reused in-memory before the middleware queries the database again.
	// A short TTL keeps agent status (ONLINE/OFFLINE) reasonably fresh while
	// still removing the per-request DB lookup from the auth hot path.
	agentAuthCacheTTL = 5 * time.Minute

	// agentAuthCacheCleanupInterval is how often the background sweeper
	// evicts expired entries. Chosen to be larger than the TTL so most
	// evictions happen lazily on read; the sweep only catches orphans that
	// were never read again.
	agentAuthCacheCleanupInterval = 10 * time.Minute
)

// agentCacheEntry holds a snapshot of a validated agent and the time at which
// the snapshot should be considered stale. The agent is stored by value to
// insulate the cache from downstream mutation; callers receive a fresh copy
// from get.
type agentCacheEntry struct {
	agent     domain.Agent
	expiresAt time.Time
}

// agentAuthCache is a process-wide, concurrency-safe TTL cache for validated
// agent lookups. It is keyed by the agent's runtime secret, which is the only
// identifier available on an auth request before a database roundtrip.
//
// The cache stores a value copy of the agent on write and a fresh value copy
// on read, so external code mutating the *domain.Agent it receives from
// gin.Context cannot poison subsequent cache hits.
//
// A single background goroutine, started exactly once via sync.Once, evicts
// expired entries on a fixed interval.
type agentAuthCache struct {
	entries sync.Map // map[string]agentCacheEntry
	ttl     time.Duration

	startCleanup sync.Once
	stopCleanup  chan struct{}
}

var (
	// defaultAgentAuthCache is the process-wide singleton used by AgentAuth.
	// It is created lazily on first use and shared by every middleware
	// instance, so the cleanup goroutine never starts more than once even
	// if AgentAuth() is invoked multiple times.
	defaultAgentAuthCache     *agentAuthCache
	defaultAgentAuthCacheOnce sync.Once
)

// newAgentAuthCache constructs a cache with the given TTL. If cleanupInterval
// is positive, a single background goroutine is started that evicts expired
// entries on every tick. The goroutine can be stopped via Stop().
func newAgentAuthCache(ttl, cleanupInterval time.Duration) *agentAuthCache {
	c := &agentAuthCache{
		ttl:         ttl,
		stopCleanup: make(chan struct{}),
	}
	if cleanupInterval > 0 {
		c.startCleanup.Do(func() {
			go c.runCleanup(cleanupInterval)
		})
	}
	return c
}

// get returns a fresh copy of the cached agent when present and not expired.
// The second return value reports whether the cache produced a usable entry.
// Expired entries are removed lazily as a side effect.
func (c *agentAuthCache) get(secret string) (*domain.Agent, bool) {
	raw, ok := c.entries.Load(secret)
	if !ok {
		return nil, false
	}

	entry, ok := raw.(agentCacheEntry)
	if !ok {
		// Defensive: an unexpected value type would otherwise wedge this
		// secret in the map forever. Drop it and treat as a miss.
		c.entries.Delete(secret)
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		c.entries.Delete(secret)
		return nil, false
	}

	// Return a copy so callers cannot mutate the cached record.
	cp := entry.agent
	return &cp, true
}

// set stores a snapshot of agent in the cache under secret for the configured
// TTL. Nil agents and empty secrets are ignored; this prevents the cache from
// retaining "not found" results and from polluting itself with empty keys.
func (c *agentAuthCache) set(secret string, agent *domain.Agent) {
	if agent == nil || secret == "" {
		return
	}
	c.entries.Store(secret, agentCacheEntry{
		agent:     *agent,
		expiresAt: time.Now().Add(c.ttl),
	})
}

// runCleanup periodically scans the cache and removes expired entries.
// Missing or malformed entries are also dropped to keep the map healthy.
func (c *agentAuthCache) runCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCleanup:
			return
		case now := <-ticker.C:
			c.entries.Range(func(key, value any) bool {
				entry, ok := value.(agentCacheEntry)
				if !ok || now.After(entry.expiresAt) {
					c.entries.Delete(key)
				}
				return true
			})
		}
	}
}

// Stop terminates the background cleanup goroutine. It is safe to call
// multiple times. Intended for graceful shutdown and tests.
func (c *agentAuthCache) Stop() {
	select {
	case <-c.stopCleanup:
		// already closed
	default:
		close(c.stopCleanup)
	}
}

// currentAgentAuthCache returns the process-wide singleton cache, creating
// it on first call. If a cache has already been assigned to the package
// variable (e.g. by tests injecting a custom TTL), that instance is returned
// and left untouched.
func currentAgentAuthCache() *agentAuthCache {
	defaultAgentAuthCacheOnce.Do(func() {
		if defaultAgentAuthCache == nil {
			defaultAgentAuthCache = newAgentAuthCache(agentAuthCacheTTL, agentAuthCacheCleanupInterval)
		}
	})
	return defaultAgentAuthCache
}
