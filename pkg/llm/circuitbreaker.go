package llm

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned (fail-fast, no provider cost) while the circuit
// breaker is open. Callers should treat it like a retryable transport error
// and fail over to an alternate model when one is configured.
var ErrCircuitOpen = errors.New("llm: circuit breaker open")

type breakerState int

const (
	breakerClosed breakerState = iota
	breakerOpen
	breakerHalfOpen
)

// circuitBreaker trips after threshold consecutive provider failures and
// stays open for cooldown, then admits a single half-open trial. A successful
// trial closes the circuit; a failed trial re-opens it. Context cancellations
// never count as provider failures.
type circuitBreaker struct {
	mu          sync.Mutex
	threshold   int
	cooldown    time.Duration
	failures    int
	state       breakerState
	openedAt    time.Time
	trialActive bool
}

func newCircuitBreaker(threshold int, cooldown time.Duration) *circuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &circuitBreaker{threshold: threshold, cooldown: cooldown}
}

// beforeCall reports whether a call may proceed. The returned bool is true
// when this call is the half-open trial (its outcome decides the circuit).
func (b *circuitBreaker) beforeCall() (allow bool, isTrial bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case breakerClosed:
		return true, false
	case breakerOpen:
		if time.Since(b.openedAt) >= b.cooldown {
			b.state = breakerHalfOpen
			b.trialActive = true
			return true, true
		}
		return false, false
	default: // breakerHalfOpen
		if b.trialActive {
			return false, false
		}
		b.trialActive = true
		return true, true
	}
}

func (b *circuitBreaker) afterCall(trial bool, err error) {
	if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, ErrCircuitOpen)) {
		if trial {
			b.mu.Lock()
			b.trialActive = false
			b.mu.Unlock()
		}
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err == nil {
		b.failures = 0
		b.state = breakerClosed
		b.trialActive = false
		return
	}
	b.failures++
	if b.state == breakerHalfOpen || b.failures >= b.threshold {
		b.state = breakerOpen
		b.openedAt = time.Now()
	}
	b.trialActive = false
}

// State returns "closed", "open", or "half-open" for observability.
func (b *circuitBreaker) State() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case breakerOpen:
		return "open"
	case breakerHalfOpen:
		return "half-open"
	default:
		return "closed"
	}
}

// WithCircuitBreaker trips after threshold consecutive provider failures and
// fast-fails for cooldown before admitting a half-open trial. Example:
// llm.WrapClient(raw, llm.WithCircuitBreaker(5, 30*time.Second)).
func WithCircuitBreaker(threshold int, cooldown time.Duration) MiddlewareOption {
	return func(mc *MiddlewareClient) {
		mc.breaker = newCircuitBreaker(threshold, cooldown)
	}
}

// BreakerState exposes the wrapped circuit state ("closed"/"open"/"half-open",
// or "disabled" when no breaker is configured).
func (mc *MiddlewareClient) BreakerState() string {
	if mc.breaker == nil {
		return "disabled"
	}
	return mc.breaker.State()
}
