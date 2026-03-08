package tasks

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Validator handles JSON repair and schema validation for LLM outputs.
type Validator struct{}

// RepairJSON attempts to fix common LLM formatting issues (e.g., markdown blocks).
func (v *Validator) RepairJSON(input string) string {
	cleaned := strings.TrimSpace(input)

	// 1. Remove Markdown code blocks if present
	re := regexp.MustCompile("(?s)```(?:json)?\n?(.*?)\n?```")
	if matches := re.FindStringSubmatch(cleaned); len(matches) > 1 {
		cleaned = strings.TrimSpace(matches[1])
	}

	return cleaned
}

// ValidateSchema ensures the repaired JSON can be unmarshaled into the target schema.
func (v *Validator) ValidateSchema(jsonStr string, schema interface{}) (interface{}, error) {
	if schema == nil {
		return jsonStr, nil
	}

	// Unmarshal into the target schema
	err := json.Unmarshal([]byte(jsonStr), schema)
	if err != nil {
		return nil, fmt.Errorf("JSON does not match schema: %w", err)
	}

	return schema, nil
}
