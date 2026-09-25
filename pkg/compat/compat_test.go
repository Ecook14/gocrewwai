package compat

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestConvertValue tests the ConvertValue function from compat.go.
func TestConvertValue(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  interface{}
	}{
		{
			name:  "string passthrough",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "int passthrough",
			input: 42,
			want:  42,
		},
		{
			name:  "float passthrough",
			input: 3.14,
			want:  3.14,
		},
		{
			name:  "bool passthrough",
			input: true,
			want:  true,
		},
		{
			name:  "nil passthrough",
			input: nil,
			want:  nil,
		},
		{
			name:  "slice of strings",
			input: []any{"a", "b", "c"},
			want:  []any{"a", "b", "c"},
		},
		{
			name:  "map of strings",
			input: map[string]any{"key": "value"},
			want:  map[string]any{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertValue(tt.input)
			if result == nil && tt.want == nil {
				return
			}
			if result == nil || tt.want == nil {
				t.Errorf("ConvertValue(%v) = %v, want %v", tt.input, result, tt.want)
				return
			}
			// Use deep equality for slices/maps since they're not comparable with ==
			if !deepEqual(result, tt.want) {
				t.Errorf("ConvertValue(%v) = %v, want %v", tt.input, result, tt.want)
			}
		})
	}
}

func deepEqual(a, b interface{}) bool {
	ar := reflect.ValueOf(a)
	br := reflect.ValueOf(b)
	if ar.Kind() != br.Kind() {
		// Allow cross-kind numeric comparison (e.g. int vs int64 from ConvertValue)
		if isNumericKind(ar.Kind()) && isNumericKind(br.Kind()) {
			return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
		}
		return false
	}
	switch ar.Kind() {
	case reflect.Slice, reflect.Map:
		aJSON, _ := json.Marshal(a)
		bJSON, _ := json.Marshal(b)
		return string(aJSON) == string(bJSON)
	default:
		return a == b
	}
}

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// TestMessageFromGoc tests message conversion.
func TestMessageFromGoc(t *testing.T) {
	msg := MessageFromGoc("user", "Hello world")
	if msg["role"] != "user" {
		t.Errorf("MessageFromGoc role = %v, want %v", msg["role"], "user")
	}
	if msg["content"] != "Hello world" {
		t.Errorf("MessageFromGoc content = %v, want %v", msg["content"], "Hello world")
	}
}

// TestContentFromMessages tests message-to-content conversion.
func TestContentFromMessages(t *testing.T) {
	messages := []map[string]interface{}{
		{"role": "user", "content": "Hello"},
		{"role": "assistant", "content": "Hi"},
	}
	content := ContentFromMessages(messages)
	if content == "" {
		t.Error("ContentFromMessages should not return empty for non-nil input")
	}
}

// TestContentFromMessagesNil tests nil handling.
func TestContentFromMessagesNil(t *testing.T) {
	content := ContentFromMessages(nil)
	if content != "" {
		t.Errorf("ContentFromMessages(nil) = %q, want empty", content)
	}
}

// TestContextWithTimeout tests context creation.
func TestContextWithTimeout(t *testing.T) {
	ctx, cancel := ContextWithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("Context should have deadline")
	}
	if time.Until(deadline) > 5*time.Second {
		t.Errorf("Context deadline too far: %v", time.Until(deadline))
	}
}

// TestContextWithCancel tests cancellable context.
func TestContextWithCancel(t *testing.T) {
	ctx, cancel := ContextWithCancel(context.Background())
	cancel()

	select {
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			t.Errorf("ctx.Err() = %v, want %v", ctx.Err(), context.Canceled)
		}
	default:
		t.Error("Cancelled context should be ready on Done channel")
	}
}

// TestContextWithValue tests context value storage.
func TestContextWithValue(t *testing.T) {
	ctx := ContextWithValue(context.Background(), "test-key", "test-value")
	if val := ctx.Value("test-key"); val != "test-value" {
		t.Errorf("Context value = %v, want %v", val, "test-value")
	}
}

// TestFormatError tests error formatting.
func TestFormatError(t *testing.T) {
	err := FormatError("something went wrong: %s", "test")
	if err == nil {
		t.Fatal("FormatError returned nil")
	}
	expected := "something went wrong: test"
	if err.Error() != expected {
		t.Errorf("FormatError() = %q, want %q", err.Error(), expected)
	}
}

// TestWrapError tests error wrapping.
func TestWrapError(t *testing.T) {
	inner := fmt.Errorf("inner error")
	wrapped := WrapError(inner, "wrapper")
	if wrapped == nil {
		t.Fatal("WrapError returned nil")
	}
	if !strings.Contains(fmt.Sprint(wrapped), "wrapper") {
		t.Errorf("WrapError should contain wrapper context")
	}
}

// TestIsCanceled tests cancellation detection.
func TestIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ctx.Err()
	if !IsCanceled(err) {
		t.Errorf("IsCanceled(%v) should return true", err)
	}
}

// TestIsTimeout tests timeout detection.
func TestIsTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	<-ctx.Done()
	err := ctx.Err()
	if !IsTimeout(err) {
		t.Errorf("IsTimeout(%v) should return true", err)
	}
}

// TestIsNotFound tests not-found detection.
func TestIsNotFound(t *testing.T) {
	err := fmt.Errorf("key not found")
	if !IsNotFound(err) {
		t.Errorf("IsNotFound(%v) should return true", err)
	}
}

// TestIsConflict tests conflict detection.
func TestIsConflict(t *testing.T) {
	err := fmt.Errorf("already exists")
	if !IsConflict(err) {
		t.Errorf("IsConflict(%v) should return true", err)
	}
}

// TestIsUnauthorized tests unauthorized detection.
func TestIsUnauthorized(t *testing.T) {
	err := fmt.Errorf("permission denied")
	if !IsUnauthorized(err) {
		t.Errorf("IsUnauthorized(%v) should return true", err)
	}
}

// TestIsRateLimited tests rate limit detection.
func TestIsRateLimited(t *testing.T) {
	err := fmt.Errorf("rate limit exceeded")
	if !IsRateLimited(err) {
		t.Errorf("IsRateLimited(%v) should return true", err)
	}
}
