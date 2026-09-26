package llm

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type failNTimes struct {
	mu        sync.Mutex
	remaining int
	err       error
}

func (f *failNTimes) Generate(ctx context.Context, messages []Message, options GenerateOptions) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.remaining > 0 {
		f.remaining--
		return "", f.err
	}
	return "ok", nil
}

func (f *failNTimes) GenerateWithUsage(ctx context.Context, messages []Message, options GenerateOptions) (string, *Usage, error) {
	out, err := f.Generate(ctx, messages, options)
	if err != nil {
		return "", nil, err
	}
	return out, &Usage{}, nil
}

func (f *failNTimes) GenerateStructured(ctx context.Context, messages []Message, schema interface{}, options GenerateOptions) (interface{}, error) {
	out, err := f.Generate(ctx, messages, options)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (f *failNTimes) StreamGenerate(ctx context.Context, messages []Message, options GenerateOptions) (<-chan string, error) {
	_, err := f.Generate(ctx, messages, options)
	if err != nil {
		return nil, err
	}
	ch := make(chan string, 1)
	ch <- "ok"
	close(ch)
	return ch, nil
}

var errProvider = errors.New("provider down")

func TestBreakerTripsAndFastFails(t *testing.T) {
	inner := &failNTimes{remaining: 100, err: errProvider}
	mc := WrapClient(inner, WithCircuitBreaker(3, time.Minute))
	for i := 0; i < 3; i++ {
		if _, err := mc.Generate(context.Background(), nil, GenerateOptions{}); !errors.Is(err, errProvider) {
			t.Fatalf("call %d: got %v, want provider error", i, err)
		}
	}
	if st := mc.BreakerState(); st != "open" {
		t.Fatalf("state = %s, want open", st)
	}
	if _, err := mc.Generate(context.Background(), nil, GenerateOptions{}); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("fast-fail: got %v, want ErrCircuitOpen", err)
	}
}

func TestBreakerHalfOpenRecovery(t *testing.T) {
	inner := &failNTimes{remaining: 2, err: errProvider}
	mc := WrapClient(inner, WithCircuitBreaker(2, 20*time.Millisecond))
	for i := 0; i < 2; i++ {
		mc.Generate(context.Background(), nil, GenerateOptions{})
	}
	if _, err := mc.Generate(context.Background(), nil, GenerateOptions{}); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("want open, got %v", err)
	}
	time.Sleep(40 * time.Millisecond)
	out, err := mc.Generate(context.Background(), nil, GenerateOptions{})
	if err != nil || out != "ok" {
		t.Fatalf("trial: got %q, %v", out, err)
	}
	if st := mc.BreakerState(); st != "closed" {
		t.Fatalf("state = %s, want closed", st)
	}
}

func TestBreakerSuccessResetsCount(t *testing.T) {
	inner := &failNTimes{remaining: 1, err: errProvider}
	mc := WrapClient(inner, WithCircuitBreaker(3, time.Minute))
	mc.Generate(context.Background(), nil, GenerateOptions{})
	mc.Generate(context.Background(), nil, GenerateOptions{})
	mc.Generate(context.Background(), nil, GenerateOptions{})
	if st := mc.BreakerState(); st != "closed" {
		t.Fatalf("state = %s, want closed after successes", st)
	}
}

func TestBreakerIgnoresCancel(t *testing.T) {
	inner := &failNTimes{remaining: 100, err: context.Canceled}
	mc := WrapClient(inner, WithCircuitBreaker(2, time.Minute))
	for i := 0; i < 5; i++ {
		mc.Generate(context.Background(), nil, GenerateOptions{})
	}
	if st := mc.BreakerState(); st != "closed" {
		t.Fatalf("cancels tripped breaker: %s", st)
	}
}

func TestBreakerDisabledByDefault(t *testing.T) {
	mc := WrapClient(&failNTimes{})
	if st := mc.BreakerState(); st != "disabled" {
		t.Fatalf("state = %s, want disabled", st)
	}
}
