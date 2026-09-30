package memory

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("bad fixture time: %v", err)
	}
	return ts
}

func seedScopes(t *testing.T, s Store) {
	t.Helper()
	ctx := context.Background()
	items := []struct {
		id, scope string
	}{
		{"1", "/tenant/a"},
		{"2", "/tenant/a/project"},
		{"3", "/tenant/ab"},
		{"4", "/other"},
	}
	for _, it := range items {
		if err := s.Add(ctx, &MemoryItem{
			ID:       it.id,
			Text:     "t",
			Metadata: map[string]interface{}{"scope": it.scope},
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
}

func TestInMemDeleteScope(t *testing.T) {
	s := NewInMemCosineStore()
	seedScopes(t, s)
	n, err := s.DeleteScope(context.Background(), "/tenant/a")
	if err != nil {
		t.Fatalf("DeleteScope: %v", err)
	}
	if n != 2 {
		t.Errorf("deleted = %d, want 2 (/tenant/a + descendant, not sibling /tenant/ab)", n)
	}
	left, _ := s.Count(context.Background())
	if left != 2 {
		t.Errorf("remaining = %d, want 2", left)
	}
}

func TestSQLiteDeleteScope(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "scope_test.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	defer s.Close()
	seedScopes(t, s)
	n, err := s.DeleteScope(context.Background(), "/tenant/a")
	if err != nil {
		t.Fatalf("DeleteScope: %v", err)
	}
	if n != 2 {
		t.Errorf("deleted = %d, want 2", n)
	}
	left, _ := s.Count(context.Background())
	if left != 2 {
		t.Errorf("remaining = %d, want 2", left)
	}
}

func TestSQLiteExpiryInstantCompare(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "expiry_test.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	defer s.Close()
	ctx := context.Background()
	past := "2000-01-01T00:00:00Z"
	if err := s.Add(ctx, &MemoryItem{ID: "old", Text: "t", ExpiresAt: mustParseTime(t, past)}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.PurgeExpired(ctx); err != nil {
		t.Fatalf("purge: %v", err)
	}
	n, _ := s.Count(ctx)
	if n != 0 {
		t.Errorf("expired record survived purge/count: %d", n)
	}
}
