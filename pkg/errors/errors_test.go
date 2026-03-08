package errors

import (
	"fmt"
	"testing"
)

func TestAgentError_Corruption(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected string
	}{
		{"16 chars", "Elite Researcher", "agent 'Elite Researcher' (iter 1): fail"},
		{"17 chars", "Technical Blogger", "agent 'Technical Blogger' (iter 1): fail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAgentError(tt.role, 1, fmt.Errorf("fail"))
			got := err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}
