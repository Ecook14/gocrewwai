package tasks

import (
	"testing"
)

func TestValidator_RepairJSON(t *testing.T) {
	v := &Validator{}
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean JSON",
			input:    `{"test": "value"}`,
			expected: `{"test": "value"}`,
		},
		{
			name:     "Markdown JSON Block",
			input:    "```json\n{\"test\": \"value\"}\n```",
			expected: `{"test": "value"}`,
		},
		{
			name:     "Markdown Block without type",
			input:    "```\n{\"test\": \"value\"}\n```",
			expected: `{"test": "value"}`,
		},
		{
			name:     "Messy formatting",
			input:    "   ```json\n\n  {\"test\": \"value\"}  \n\n```   ",
			expected: `{"test": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.RepairJSON(tt.input); got != tt.expected {
				t.Errorf("RepairJSON() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidator_ValidateSchema(t *testing.T) {
	v := &Validator{}
	
	type TestSchema struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	t.Run("Valid Schema", func(t *testing.T) {
		schema := &TestSchema{}
		jsonStr := `{"name": "John", "age": 30}`
		_, err := v.ValidateSchema(jsonStr, schema)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if schema.Name != "John" || schema.Age != 30 {
			t.Errorf("Schema not populated correctly: %+v", schema)
		}
	})

	t.Run("Invalid Schema", func(t *testing.T) {
		schema := &TestSchema{}
		jsonStr := `{"name": "John", "age": "thirty"}` // age should be int
		_, err := v.ValidateSchema(jsonStr, schema)
		if err == nil {
			t.Error("Expected error for invalid types")
		}
	})
}
