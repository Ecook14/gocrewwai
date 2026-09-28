package memory

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryPolicy configures transient-failure retries for remote stores.
type RetryPolicy struct {
	// Attempts is the total number of tries (1 = no retry). Clamped to [1,10].
	Attempts int
	// BaseBackoff is the initial delay, doubled each retry up to MaxBackoff.
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
}

// DefaultRetryPolicy is 3 attempts, 100ms base, 2s max.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{Attempts: 3, BaseBackoff: 100 * time.Millisecond, MaxBackoff: 2 * time.Second}
}

func (p RetryPolicy) normalized() RetryPolicy {
	if p.Attempts < 1 {
		p.Attempts = 1
	}
	if p.Attempts > 10 {
		p.Attempts = 10
	}
	if p.BaseBackoff <= 0 {
		p.BaseBackoff = 100 * time.Millisecond
	}
	if p.MaxBackoff <= 0 || p.MaxBackoff < p.BaseBackoff {
		p.MaxBackoff = 2 * time.Second
	}
	return p
}

// RetryStore decorates any Store, retrying transient failures with exponential
// backoff and jitter-free doubling. Context cancellation aborts immediately
// and is never retried. Reset/Delete are not retried (non-idempotent risk).
type RetryStore struct {
	inner  Store
	policy RetryPolicy
}

// WithRetry wraps store with the given retry policy.
func WithRetry(store Store, policy RetryPolicy) *RetryStore {
	return &RetryStore{inner: store, policy: policy.normalized()}
}

// Unwrap returns the underlying store.
func (r *RetryStore) Unwrap() Store { return r.inner }

func (r *RetryStore) do(ctx context.Context, op string, fn func(context.Context) error) error {
	var err error
	backoff := r.policy.BaseBackoff
	for attempt := 1; attempt <= r.policy.Attempts; attempt++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err = fn(ctx); err == nil {
			return nil
		}
		if attempt == r.policy.Attempts {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff = time.Duration(math.Min(float64(backoff*2), float64(r.policy.MaxBackoff)))
	}
	return fmt.Errorf("memory: %s failed after %d attempts: %w", op, r.policy.Attempts, err)
}

func (r *RetryStore) Add(ctx context.Context, item *MemoryItem) error {
	return r.do(ctx, "add", func(ctx context.Context) error { return r.inner.Add(ctx, item) })
}

func (r *RetryStore) BulkAdd(ctx context.Context, items []*MemoryItem) error {
	return r.do(ctx, "bulk-add", func(ctx context.Context) error { return r.inner.BulkAdd(ctx, items) })
}

func (r *RetryStore) Delete(ctx context.Context, id string) error {
	return r.inner.Delete(ctx, id)
}

func (r *RetryStore) Search(ctx context.Context, queryVector []float32, limit int) ([]*MemoryItem, error) {
	var out []*MemoryItem
	err := r.do(ctx, "search", func(ctx context.Context) error {
		var err error
		out, err = r.inner.Search(ctx, queryVector, limit)
		return err
	})
	return out, err
}

func (r *RetryStore) Count(ctx context.Context) (int, error) {
	var n int
	err := r.do(ctx, "count", func(ctx context.Context) error {
		var err error
		n, err = r.inner.Count(ctx)
		return err
	})
	return n, err
}

func (r *RetryStore) Reset(ctx context.Context) error {
	return r.inner.Reset(ctx)
}
