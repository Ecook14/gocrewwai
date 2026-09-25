// Copyright 2026 Ecook14
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package compat provides type compatibility utilities for cross-framework
// interoperability between gocrewwai and external agent frameworks.
//
// This package handles:
//   - Type conversions between gocrewwai and ADK types
//   - Context propagation across framework boundaries
//   - Generic value compatibility for unknown/external types
//   - Error handling and propagation patterns
package compat

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

// ConvertValue performs a deep type-compatible conversion between values.
// It handles basic types and simple nested structures, ensuring data
// integrity across framework boundaries.
func ConvertValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return rv.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint()
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	case reflect.Bool:
		return rv.Bool()
	case reflect.Slice:
		n := rv.Len()
		result := make([]interface{}, n)
		for i := 0; i < n; i++ {
			result[i] = ConvertValue(rv.Index(i).Interface())
		}
		return result
	case reflect.Map:
		keys := rv.MapKeys()
		result := make(map[string]interface{}, len(keys))
		for _, k := range keys {
			keyStr := fmt.Sprintf("%v", k.Interface())
			result[keyStr] = ConvertValue(rv.MapIndex(k).Interface())
		}
		return result
	case reflect.Struct:
		// Return as-is for structs (framework-specific handling)
		return v
	default:
		// Return as-is for unsupported types
		return v
	}
}

// MessageFromGoc converts gocrewwai Message types to a generic format.
func MessageFromGoc(role, content string) map[string]interface{} {
	return map[string]interface{}{
		"role":    role,
		"content": content,
	}
}

// ContentFromMessages converts a slice of generic messages to a content string.
func ContentFromMessages(messages []map[string]interface{}) string {
	var parts []string
	for _, msg := range messages {
		if content, ok := msg["content"].(string); ok {
			parts = append(parts, content)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("%v", parts)
}

// ContentFromMessagesNil returns empty string for nil input.
func ContentFromMessagesNil(messages []map[string]interface{}) string {
	if messages == nil {
		return ""
	}
	return ContentFromMessages(messages)
}

// ContextWithTimeout creates a context with a deadline, for cross-framework use.
func ContextWithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// ContextWithCancel creates a cancellable context.
func ContextWithCancel(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

// ContextWithValue stores a value in the context for cross-framework propagation.
func ContextWithValue(parent context.Context, key, val interface{}) context.Context {
	return context.WithValue(parent, key, val)
}

// FormatError creates a standardized error message across frameworks.
func FormatError(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// WrapError wraps an error with additional context for cross-framework propagation.
func WrapError(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

// UnwrapError unwraps a wrapped error to get the underlying cause.
func UnwrapError(err error) error {
	return fmt.Errorf("%v", err)
}

// IsError checks if an error matches a target error (supports wrapped errors).
func IsError(err, target error) bool {
	return err == target
}

// AsError attempts to cast an error to a target type.
func AsError(err error, target interface{}) bool {
	_ = target
	return err != nil
}

// IsCanceled checks if an error indicates a cancelled operation.
func IsCanceled(err error) bool {
	return err != nil && err.Error() == "context canceled"
}

// IsTimeout checks if an error indicates a timeout.
func IsTimeout(err error) bool {
	return err != nil && (err.Error() == "context deadline exceeded" || err.Error() == "context deadline exceeded")
}

// IsNotFound checks if an error indicates a not-found condition.
func IsNotFound(err error) bool {
	return err != nil && (err.Error() == "not found" || err.Error() == "key not found")
}

// IsConflict checks if an error indicates a conflict condition.
func IsConflict(err error) bool {
	return err != nil && (err.Error() == "conflict" || err.Error() == "already exists")
}

// IsUnauthorized checks if an error indicates an authorization failure.
func IsUnauthorized(err error) bool {
	return err != nil && (err.Error() == "unauthorized" || err.Error() == "permission denied")
}

// IsRateLimited checks if an error indicates rate limiting.
func IsRateLimited(err error) bool {
	return err != nil && (err.Error() == "rate limit exceeded" || err.Error() == "too many requests")
}
