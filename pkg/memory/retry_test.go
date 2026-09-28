package memory

import (
	"context"
	"errors"
	"testing"
	"time"
)

type flakyStore struct {
	failures int
	calls    int
}

func (f *flakyStore) Add(ctx context.Context, item *MemoryItem) error {
	f.calls++
	if f.calls <= f.failures {
		return errors.New("transient")
	}
	return nil
}
func (f *flakyStore) BulkAdd(ctx context.Context, items []*MemoryItem) error { return nil }
func (f *flakyStore) Delete(ctx context.Context, id string) error            { return nil }
func (f *flakyStore) Search(ctx context.Context, q []float32, limit int) ([]*MemoryItem, error) {
	return nil, nil
}
func (f *flakyStore) Count(ctx context.Context) (int, error) { return 0, nil }
func (f *flakyStore) Reset(ctx context.Context) error        { return nil }

func TestRetryStore_SucceedsAfterTransient(t *testing.T) {
	inner := &flakyStore{failures: 2}
	r := WithRetry(inner, RetryPolicy{Attempts: 3, BaseBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond})
	if err := r.Add(context.Background(), &MemoryItem{}); err != nil {
		t.Fatalf("expected success after retries: %v", err)
	}
	if inner.calls != 3 {
		t.Errorf("calls = %d, want 3", inner.calls)
	}
}

func TestRetryStore_GivesUp(t *testing.T) {
	inner := &flakyStore{failures: 10}
	r := WithRetry(inner, RetryPolicy{Attempts: 2, BaseBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond})
	if err := r.Add(context.Background(), &MemoryItem{}); err == nil {
		t.Error("expected error after exhausting attempts")
	}
	if inner.calls != 2 {
		t.Errorf("calls = %d, want 2", inner.calls)
	}
}

func TestRetryStore_CtxCancel(t *testing.T) {
	inner := &flakyStore{failures: 10}
	r := WithRetry(inner, RetryPolicy{Attempts: 5, BaseBackoff: time.Hour, MaxBackoff: time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.Add(ctx, &MemoryItem{}); !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if inner.calls != 0 {
		t.Errorf("calls = %d, want 0 (no attempt on cancelled ctx)", inner.calls)
	}
}
