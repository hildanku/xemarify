package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/agent/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testDB holds connection details for the integration test database.
// Adjust these to match your local / CI environment.
const (
	testDBHost     = "localhost"
	testDBPort     = 5445
	testDBUser     = "xemarify_manager"
	testDBPassword = "xemarify_manager"
	testDBName     = "xemarify_manager"
	testDBSSLMode  = "disable"
)

// newTestPool creates a pgxpool connection for tests and skips the test if the
// database is not reachable.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		testDBUser,
		testDBPassword,
		testDBHost,
		testDBPort,
		testDBName,
		testDBSSLMode,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping integration test – could not create pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping integration test – database unreachable: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

// newTestAgent returns a valid Agent with a unique key so tests don't collide.
// NOTE: ip_address is stored as PostgreSQL inet; the value is returned as
// CIDR notation (e.g. "192.168.1.100/32") when cast to text.
func newTestAgent() *domain.Agent {
	return &domain.Agent{
		ID:        uuid.New(),
		Name:      "test-agent-" + uuid.New().String()[:8],
		Hostname:  "host.local",
		Key:       uuid.New().String(),
		IPAddress: "192.168.1.100",
		Version:   "1.0.0",
		Status:    domain.AgentStatusOffline,
	}
}

// inetText converts a plain IP address into the text representation that
// PostgreSQL returns for an inet column (host address + /prefix).
func inetText(ip string) string {
	if ip == "" {
		return ""
	}
	// IPv4: append /32 if no prefix already present.
	for _, c := range ip {
		if c == '/' {
			return ip
		}
	}
	return ip + "/32"
}

func TestPgAgentRepository_Create(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPgAgentRepository(pool)
	ctx := context.Background()

	agent := newTestAgent()

	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	// Cleanup – remove the row so this test is idempotent.
	t.Cleanup(func() {
		_ = repo.Delete(ctx, agent.ID)
	})
}

func TestPgAgentRepository_Create_Duplicate(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPgAgentRepository(pool)
	ctx := context.Background()

	agent := newTestAgent()

	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("first Create: unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, agent.ID) })

	// Inserting the same ID a second time must fail (PK violation).
	if err := repo.Create(ctx, agent); err == nil {
		t.Fatal("second Create with duplicate ID: expected error, got nil")
	}
}

func TestPgAgentRepository_GetByID(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPgAgentRepository(pool)
	ctx := context.Background()

	agent := newTestAgent()
	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, agent.ID) })

	got, err := repo.GetByID(ctx, agent.ID)
	if err != nil {
		t.Fatalf("GetByID: unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID: expected agent, got nil")
	}
	if got.ID != agent.ID {
		t.Errorf("GetByID: ID mismatch: want %s, got %s", agent.ID, got.ID)
	}
	if got.Name != agent.Name {
		t.Errorf("GetByID: Name mismatch: want %q, got %q", agent.Name, got.Name)
	}
	if got.Key != agent.Key {
		t.Errorf("GetByID: Key mismatch: want %q, got %q", agent.Key, got.Key)
	}
	if got.IPAddress != inetText(agent.IPAddress) {
		t.Errorf("GetByID: IPAddress mismatch: want %q, got %q", inetText(agent.IPAddress), got.IPAddress)
	}
}

func TestPgAgentRepository_GetByID_NotFound(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPgAgentRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetByID (not found): unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("GetByID (not found): expected nil, got %+v", got)
	}
}

func TestPgAgentRepository_GetByKey(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPgAgentRepository(pool)
	ctx := context.Background()

	agent := newTestAgent()
	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, agent.ID) })

	got, err := repo.GetByKey(ctx, agent.Key)
	if err != nil {
		t.Fatalf("GetByKey: unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("GetByKey: expected agent, got nil")
	}
	if got.ID != agent.ID {
		t.Errorf("GetByKey: ID mismatch: want %s, got %s", agent.ID, got.ID)
	}
}

func TestPgAgentRepository_GetByKey_NotFound(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPgAgentRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByKey(ctx, "non-existent-key-xyz-"+uuid.New().String())
	if err != nil {
		t.Fatalf("GetByKey (not found): unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("GetByKey (not found): expected nil, got %+v", got)
	}
}

func TestPgAgentRepository_UpdateLastSeen(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPgAgentRepository(pool)
	ctx := context.Background()

	agent := newTestAgent()
	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, agent.ID) })

	if err := repo.UpdateLastSeen(ctx, agent.ID); err != nil {
		t.Fatalf("UpdateLastSeen: unexpected error: %v", err)
	}

	// Verify status changed to ONLINE and last_seen_at is populated.
	got, err := repo.GetByID(ctx, agent.ID)
	if err != nil {
		t.Fatalf("GetByID after UpdateLastSeen: %v", err)
	}
	if got.Status != domain.AgentStatusOnline {
		t.Errorf("UpdateLastSeen: expected status ONLINE, got %q", got.Status)
	}
	if got.LastSeenAt == nil {
		t.Error("UpdateLastSeen: expected last_seen_at to be set, got nil")
	}
}

func TestPgAgentRepository_Update(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPgAgentRepository(pool)
	ctx := context.Background()

	agent := newTestAgent()
	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, agent.ID) })

	updated := &domain.Agent{
		Name:      "updated-name",
		Hostname:  "updated-host.local",
		IPAddress: "10.0.0.1",
		Version:   "2.0.0",
		Status:    domain.AgentStatusOnline,
	}

	if err := repo.Update(ctx, agent.ID, updated); err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}

	got, err := repo.GetByID(ctx, agent.ID)
	if err != nil {
		t.Fatalf("GetByID after Update: %v", err)
	}
	if got.Name != updated.Name {
		t.Errorf("Update: Name mismatch: want %q, got %q", updated.Name, got.Name)
	}
	if got.Hostname != updated.Hostname {
		t.Errorf("Update: Hostname mismatch: want %q, got %q", updated.Hostname, got.Hostname)
	}
	if got.IPAddress != inetText(updated.IPAddress) {
		t.Errorf("Update: IPAddress mismatch: want %q, got %q", inetText(updated.IPAddress), got.IPAddress)
	}
	if got.Version != updated.Version {
		t.Errorf("Update: Version mismatch: want %q, got %q", updated.Version, got.Version)
	}
	if got.Status != updated.Status {
		t.Errorf("Update: Status mismatch: want %q, got %q", updated.Status, got.Status)
	}
}

func TestPgAgentRepository_List(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPgAgentRepository(pool)
	ctx := context.Background()

	// Create two agents and track them for cleanup.
	agents := []*domain.Agent{newTestAgent(), newTestAgent()}
	for _, a := range agents {
		if err := repo.Create(ctx, a); err != nil {
			t.Fatalf("Create: %v", err)
		}
		id := a.ID
		t.Cleanup(func() { _ = repo.Delete(ctx, id) })
	}

	list, _, err := repo.List(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}

	// Build a lookup set from the returned list.
	found := make(map[uuid.UUID]bool, len(list))
	for _, a := range list {
		found[a.ID] = true
	}

	for _, a := range agents {
		if !found[a.ID] {
			t.Errorf("List: agent %s not found in results", a.ID)
		}
	}
}

func TestPgAgentRepository_Delete(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPgAgentRepository(pool)
	ctx := context.Background()

	agent := newTestAgent()
	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, agent.ID); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}

	// The agent must no longer be retrievable.
	got, err := repo.GetByID(ctx, agent.ID)
	if err != nil {
		t.Fatalf("GetByID after Delete: %v", err)
	}
	if got != nil {
		t.Fatalf("Delete: agent still exists after deletion: %+v", got)
	}
}

func TestPgAgentRepository_Delete_NonExistent(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPgAgentRepository(pool)
	ctx := context.Background()

	// Deleting a row that doesn't exist should not return an error.
	if err := repo.Delete(ctx, uuid.New()); err != nil {
		t.Fatalf("Delete (non-existent): unexpected error: %v", err)
	}
}
