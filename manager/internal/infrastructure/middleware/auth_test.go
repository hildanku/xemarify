package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/agent/domain"
	agentRepo "github.com/hildanku/xemarify/internal/modules/agent/repository"
	"github.com/sirupsen/logrus"
)

// fakeAgentRepository is a hand-rolled in-memory stand-in for the real
// AgentRepository used by the middleware tests. We intentionally avoid
// generated mocks here to keep the test self-contained.
type fakeAgentRepository struct {
	agents map[string]*domain.Agent
	calls  int32
	err    error
}

func (f *fakeAgentRepository) GetBySecret(_ context.Context, secret string) (*domain.Agent, error) {
	atomic.AddInt32(&f.calls, 1)
	if f.err != nil {
		return nil, f.err
	}
	return f.agents[secret], nil
}

// The remaining AgentRepository methods are not exercised by the middleware
// under test. They panic loudly to surface accidental use during tests.
func (f *fakeAgentRepository) CreateEnrollmentToken(context.Context, uuid.UUID, string) error {
	panic("unused")
}
func (f *fakeAgentRepository) CreateWithEnrollmentToken(context.Context, string, *domain.Agent) error {
	panic("unused")
}
func (f *fakeAgentRepository) UpdateLastSeen(context.Context, uuid.UUID) error {
	panic("unused")
}
func (f *fakeAgentRepository) Create(context.Context, *domain.Agent) error {
	panic("unused")
}
func (f *fakeAgentRepository) Update(context.Context, uuid.UUID, *domain.Agent) error {
	panic("unused")
}
func (f *fakeAgentRepository) GetByID(context.Context, uuid.UUID) (*domain.Agent, error) {
	panic("unused")
}
func (f *fakeAgentRepository) List(context.Context, agentRepo.ListFilter) ([]*domain.Agent, string, error) {
	panic("unused")
}
func (f *fakeAgentRepository) Delete(context.Context, uuid.UUID) error {
	panic("unused")
}

// installTestCache swaps the package-level singleton for a fresh cache with
// the given TTL and no background cleanup. It returns the cache so callers
// can assert on it directly. The previous cache (if any) is stopped and the
// lazy-init sync.Once is reset so the next test starts from a clean slate.
func installTestCache(t *testing.T, ttl time.Duration) *agentAuthCache {
	t.Helper()
	if defaultAgentAuthCache != nil {
		defaultAgentAuthCache.Stop()
	}
	c := newAgentAuthCache(ttl, 0) // 0 disables background cleanup
	defaultAgentAuthCache = c
	defaultAgentAuthCacheOnce = sync.Once{}
	t.Cleanup(func() {
		c.Stop()
		defaultAgentAuthCache = nil
		defaultAgentAuthCacheOnce = sync.Once{}
	})
	return c
}

func newTestRouter(repo agentRepo.AgentRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AgentAuth(repo, logrus.New()))
	r.GET("/probe", func(c *gin.Context) {
		agent := AgentFromContext(c)
		c.JSON(http.StatusOK, gin.H{
			"agent_id":   agent.ID,
			"agent_name": agent.Name,
		})
	})
	return r
}

func sendAgentRequest(t *testing.T, router *gin.Engine, secret string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	if secret != "" {
		req.Header.Set(agentSecretHeader, secret)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func newAgent(secret, name string, status domain.AgentStatus) *domain.Agent {
	return &domain.Agent{
		ID:     uuid.New(),
		Name:   name,
		Secret: secret,
		Status: status,
	}
}

func TestAgentAuth_FirstRequestQueriesDB_SecondUsesCache(t *testing.T) {
	installTestCache(t, 5*time.Minute)

	agent := newAgent("secret-abc", "agent-1", domain.AgentStatusOnline)
	repo := &fakeAgentRepository{agents: map[string]*domain.Agent{agent.Secret: agent}}
	router := newTestRouter(repo)

	rr1 := sendAgentRequest(t, router, agent.Secret)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d body=%s", rr1.Code, rr1.Body.String())
	}
	rr2 := sendAgentRequest(t, router, agent.Secret)
	if rr2.Code != http.StatusOK {
		t.Fatalf("second request: expected 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}

	if got := atomic.LoadInt32(&repo.calls); got != 1 {
		t.Fatalf("expected exactly 1 DB call across two requests, got %d", got)
	}
}

func TestAgentAuth_ExpiredEntryTriggersDBLookup(t *testing.T) {
	cache := installTestCache(t, 10*time.Millisecond)

	agent := newAgent("secret-exp", "agent-2", domain.AgentStatusOnline)
	repo := &fakeAgentRepository{agents: map[string]*domain.Agent{agent.Secret: agent}}
	router := newTestRouter(repo)

	if rr := sendAgentRequest(t, router, agent.Secret); rr.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rr.Code)
	}
	if got := atomic.LoadInt32(&repo.calls); got != 1 {
		t.Fatalf("after first request: expected 1 DB call, got %d", got)
	}

	// Wait past the TTL and confirm the next request hits the DB again.
	time.Sleep(30 * time.Millisecond)
	if rr := sendAgentRequest(t, router, agent.Secret); rr.Code != http.StatusOK {
		t.Fatalf("post-expiry request: expected 200, got %d", rr.Code)
	}
	if got := atomic.LoadInt32(&repo.calls); got != 2 {
		t.Fatalf("after expiry: expected 2 DB calls, got %d", got)
	}

	// Sanity: after the post-expiry request the middleware re-populated the
	// cache, so the entry must be present and not yet expired.
	raw, ok := cache.entries.Load(agent.Secret)
	if !ok {
		t.Fatalf("expected fresh entry to be re-cached after the post-expiry request")
	}
	entry, ok := raw.(agentCacheEntry)
	if !ok {
		t.Fatalf("unexpected cache value type %T", raw)
	}
	if time.Now().After(entry.expiresAt) {
		t.Fatalf("re-cached entry is already expired: expiresAt=%v now=%v", entry.expiresAt, time.Now())
	}
}

func TestAgentAuth_OfflineAgentRejected(t *testing.T) {
	installTestCache(t, 5*time.Minute)

	offline := newAgent("secret-off", "agent-off", domain.AgentStatusOffline)
	repo := &fakeAgentRepository{agents: map[string]*domain.Agent{offline.Secret: offline}}
	router := newTestRouter(repo)

	rr := sendAgentRequest(t, router, offline.Secret)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for OFFLINE agent, got %d body=%s", rr.Code, rr.Body.String())
	}
	if got := atomic.LoadInt32(&repo.calls); got != 1 {
		t.Fatalf("expected 1 DB call (cache must not store OFFLINE agents), got %d", got)
	}

	// Second call should also hit DB because the OFFLINE agent was not cached.
	rr2 := sendAgentRequest(t, router, offline.Secret)
	if rr2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on second request, got %d", rr2.Code)
	}
	if got := atomic.LoadInt32(&repo.calls); got != 2 {
		t.Fatalf("expected 2 DB calls (no OFFLINE caching), got %d", got)
	}
}

func TestAgentAuth_MissingSecretRejected(t *testing.T) {
	installTestCache(t, 5*time.Minute)
	repo := &fakeAgentRepository{agents: map[string]*domain.Agent{}}
	router := newTestRouter(repo)

	rr := sendAgentRequest(t, router, "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing secret, got %d", rr.Code)
	}
	if got := atomic.LoadInt32(&repo.calls); got != 0 {
		t.Fatalf("missing-secret request must not hit DB, got %d calls", got)
	}
}

func TestAgentAuth_InvalidSecretRejected(t *testing.T) {
	installTestCache(t, 5*time.Minute)
	repo := &fakeAgentRepository{agents: map[string]*domain.Agent{}}
	router := newTestRouter(repo)

	rr := sendAgentRequest(t, router, "does-not-exist")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown secret, got %d body=%s", rr.Code, rr.Body.String())
	}
	if got := atomic.LoadInt32(&repo.calls); got != 1 {
		t.Fatalf("expected 1 DB call, got %d", got)
	}

	// Repeat with the same invalid secret. The repo is the source of truth
	// and returns nil, but we must verify the cache did not memoize the
	// "not found" result and shortcut the lookup.
	rr2 := sendAgentRequest(t, router, "does-not-exist")
	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on second request, got %d", rr2.Code)
	}
	if got := atomic.LoadInt32(&repo.calls); got != 2 {
		t.Fatalf("expected 2 DB calls (no negative caching), got %d", got)
	}
}

func TestAgentAuth_RepoErrorIs500(t *testing.T) {
	installTestCache(t, 5*time.Minute)
	repo := &fakeAgentRepository{err: errors.New("db is on fire")}
	router := newTestRouter(repo)

	rr := sendAgentRequest(t, router, "any-secret")
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on repo error, got %d", rr.Code)
	}
}

func TestAgentAuth_CacheDoesNotLeakAcrossAgents(t *testing.T) {
	installTestCache(t, 5*time.Minute)

	agentA := newAgent("secret-A", "agent-A", domain.AgentStatusOnline)
	agentB := newAgent("secret-B", "agent-B", domain.AgentStatusOnline)
	repo := &fakeAgentRepository{agents: map[string]*domain.Agent{
		agentA.Secret: agentA,
		agentB.Secret: agentB,
	}}
	router := newTestRouter(repo)

	rrA := sendAgentRequest(t, router, agentA.Secret)
	if rrA.Code != http.StatusOK {
		t.Fatalf("agent A first request: expected 200, got %d", rrA.Code)
	}
	rrB := sendAgentRequest(t, router, agentB.Secret)
	if rrB.Code != http.StatusOK {
		t.Fatalf("agent B first request: expected 200, got %d", rrB.Code)
	}

	// Two distinct secrets should each trigger exactly one DB lookup, and
	// cached entries must not be served under the wrong key.
	if got := atomic.LoadInt32(&repo.calls); got != 2 {
		t.Fatalf("expected 2 DB calls for two distinct secrets, got %d", got)
	}
}

func TestAgentAuth_CacheEntryIsCopiedNotShared(t *testing.T) {
	installTestCache(t, 5*time.Minute)

	agent := newAgent("secret-mut", "agent-mut", domain.AgentStatusOnline)
	repo := &fakeAgentRepository{agents: map[string]*domain.Agent{agent.Secret: agent}}
	router := newTestRouter(repo)

	rr1 := sendAgentRequest(t, router, agent.Secret)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rr1.Code)
	}

	// The handler exposes the agent pointer stored in gin.Context. Mutating
	// it must not change what the next request observes in the cache.
	firstAgent := agentFromResponse(t, rr1)
	firstAgent.Name = "tampered"

	rr2 := sendAgentRequest(t, router, agent.Secret)
	if rr2.Code != http.StatusOK {
		t.Fatalf("second request: expected 200, got %d", rr2.Code)
	}
	secondAgent := agentFromResponse(t, rr2)
	if secondAgent.Name == "tampered" {
		t.Fatalf("cache returned a shared pointer: external mutation leaked into the cache")
	}
	if secondAgent.Name != agent.Name {
		t.Fatalf("expected cached name %q, got %q", agent.Name, secondAgent.Name)
	}
}

// agentFromResponse decodes the agent identity from a /probe response.
// It is a test-only helper, kept private to the test file.
func agentFromResponse(t *testing.T, rr *httptest.ResponseRecorder) *domain.Agent {
	t.Helper()
	var body struct {
		AgentID   uuid.UUID `json:"agent_id"`
		AgentName string    `json:"agent_name"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return &domain.Agent{ID: body.AgentID, Name: body.AgentName}
}
