package events

import (
	"sync"
	"testing"
	"time"
)

func TestPublishSubscribe(t *testing.T) {
	b := &Bus{}
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)
	b.Publish(Event{Type: TaskStarted, Source: "test"})
	select {
	case e := <-ch:
		if e.Type != TaskStarted || e.Source != "test" {
			t.Fatalf("got %+v", e)
		}
		if e.Timestamp.IsZero() {
			t.Fatal("timestamp not auto-set")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event received")
	}
}

func TestPublishSetsTimestampWhenZero(t *testing.T) {
	b := &Bus{}
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)
	before := time.Now()
	b.Publish(Event{Type: CrewKickoffStarted})
	e := <-ch
	if e.Timestamp.Before(before) || time.Since(e.Timestamp) > time.Minute {
		t.Fatalf("bad timestamp %v", e.Timestamp)
	}
}

func TestPublishPreservesTimestamp(t *testing.T) {
	b := &Bus{}
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	b.Publish(Event{Type: TaskCompleted, Timestamp: fixed})
	if e := <-ch; !e.Timestamp.Equal(fixed) {
		t.Fatalf("timestamp overwritten: %v", e.Timestamp)
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	b := &Bus{}
	ch := b.Subscribe()
	b.Unsubscribe(ch)
	b.Publish(Event{Type: TaskFailed})
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("received event after unsubscribe")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("channel not closed on unsubscribe")
	}
}

func TestPublishFullChannelDoesNotBlock(t *testing.T) {
	b := &Bus{}
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 250; i++ {
			b.Publish(Event{Type: LLMStreamChunk})
		}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("publish blocked on full channel")
	}
}

func TestOnHandlerInvoked(t *testing.T) {
	b := &Bus{}
	var mu sync.Mutex
	var got []EventType
	done := make(chan struct{})
	b.On(func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, e.Type)
		if len(got) == 2 {
			close(done)
		}
	})
	b.Publish(Event{Type: FlowStarted})
	b.Publish(Event{Type: FlowFinished})
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handlers not invoked")
	}
}

func TestScopedHandlersCleanupCallable(t *testing.T) {
	b := &Bus{}
	cleanup := b.ScopedHandlers(func(Event) {})
	if cleanup == nil {
		t.Fatal("nil cleanup")
	}
	cleanup() // must not panic
}
