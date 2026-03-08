package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// JSONTool — JSON Processing for Agents
// ---------------------------------------------------------------------------

// JSONTool allows agents to parse, query, transform, and format JSON data.
//
// Input examples:
//
//	{"action": "parse", "data": "{\"name\": \"Alice\"}"}
//	{"action": "query", "data": "{\"users\": [{\"name\": \"Alice\"}]}", "path": "users.0.name"}
//	{"action": "format", "data": "{\"compact\":true}"}
//	{"action": "validate", "data": "{\"valid\": true}"}
//	{"action": "merge", "base": {"a": 1}, "overlay": {"b": 2}}
//	{"action": "keys", "data": "{\"a\":1,\"b\":2}"}
type JSONTool struct {
	BaseTool
}

// NewJSONTool creates a JSON processing tool.
func NewJSONTool() *JSONTool {
	return &JSONTool{
		BaseTool: BaseTool{
			NameValue:        "JSONTool",
			DescriptionValue: "Parse, query, format, validate, and transform JSON data. Actions: parse, query, format, validate, merge, keys. Input: {'action': '...', 'data': '...', 'path': '...'}",
		},
	}
}

func (t *JSONTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, _ := input["action"].(string)
	if action == "" {
		return "", fmt.Errorf("'action' is required")
	}

	switch action {
	case "parse":
		return t.parse(input)
	case "query":
		return t.query(input)
	case "format":
		return t.format(input)
	case "validate":
		return t.validate(input)
	case "merge":
		return t.merge(input)
	case "keys":
		return t.keys(input)
	default:
		return "", fmt.Errorf("unsupported action: %s", action)
	}
}

func (t *JSONTool) parse(input map[string]interface{}) (string, error) {
	data, _ := input["data"].(string)
	if data == "" {
		return "", fmt.Errorf("'data' is required")
	}
	var parsed interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}
	out, _ := json.MarshalIndent(parsed, "", "  ")
	return string(out), nil
}

func (t *JSONTool) query(input map[string]interface{}) (string, error) {
	data, _ := input["data"].(string)
	path, _ := input["path"].(string)
	if data == "" || path == "" {
		return "", fmt.Errorf("'data' and 'path' are required")
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// Navigate the path
	parts := strings.Split(path, ".")
	current := parsed
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		case []interface{}:
			var idx int
			if _, err := fmt.Sscanf(part, "%d", &idx); err == nil && idx < len(v) {
				current = v[idx]
			} else {
				return "", fmt.Errorf("invalid array index: %s", part)
			}
		default:
			return "", fmt.Errorf("cannot navigate into %T at '%s'", current, part)
		}
	}

	out, _ := json.MarshalIndent(current, "", "  ")
	return string(out), nil
}

func (t *JSONTool) format(input map[string]interface{}) (string, error) {
	data, _ := input["data"].(string)
	if data == "" {
		return "", fmt.Errorf("'data' is required")
	}
	var parsed interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}
	out, _ := json.MarshalIndent(parsed, "", "  ")
	return string(out), nil
}

func (t *JSONTool) validate(input map[string]interface{}) (string, error) {
	data, _ := input["data"].(string)
	if data == "" {
		return "", fmt.Errorf("'data' is required")
	}
	var parsed interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return fmt.Sprintf("Invalid JSON: %s", err.Error()), nil
	}
	return "Valid JSON", nil
}

func (t *JSONTool) merge(input map[string]interface{}) (string, error) {
	base, _ := input["base"].(map[string]interface{})
	overlay, _ := input["overlay"].(map[string]interface{})
	if base == nil || overlay == nil {
		return "", fmt.Errorf("'base' and 'overlay' maps are required")
	}
	merged := make(map[string]interface{})
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range overlay {
		merged[k] = v
	}
	out, _ := json.MarshalIndent(merged, "", "  ")
	return string(out), nil
}

func (t *JSONTool) keys(input map[string]interface{}) (string, error) {
	data, _ := input["data"].(string)
	if data == "" {
		return "", fmt.Errorf("'data' is required")
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return "", fmt.Errorf("data is not a JSON object: %w", err)
	}
	keys := make([]string, 0, len(parsed))
	for k := range parsed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", "), nil
}

// ---------------------------------------------------------------------------
// RegexTool — Regular Expression Processing for Agents
// ---------------------------------------------------------------------------

// RegexTool allows agents to match, extract, replace, and split text using regex.
//
// Input examples:
//
//	{"action": "match", "pattern": "\\d+", "text": "Order 12345 shipped"}
//	{"action": "findAll", "pattern": "[a-z]+@[a-z]+\\.com", "text": "contact a@b.com or c@d.com"}
//	{"action": "replace", "pattern": "\\bfoo\\b", "text": "foo bar foo", "replacement": "baz"}
//	{"action": "split", "pattern": "[,;]\\s*", "text": "a, b; c, d"}
type RegexTool struct {
	BaseTool
}

// NewRegexTool creates a regex processing tool.
func NewRegexTool() *RegexTool {
	return &RegexTool{
		BaseTool: BaseTool{
			NameValue:        "RegexTool",
			DescriptionValue: "Process text with regular expressions. Actions: match, findAll, replace, split, test. Input: {'action': '...', 'pattern': '...', 'text': '...'}",
		},
	}
}

func (t *RegexTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, _ := input["action"].(string)
	pattern, _ := input["pattern"].(string)
	text, _ := input["text"].(string)

	if action == "" || pattern == "" {
		return "", fmt.Errorf("'action' and 'pattern' are required")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	switch action {
	case "match":
		match := re.FindString(text)
		if match == "" {
			return "No match found", nil
		}
		return fmt.Sprintf("Match: %s", match), nil

	case "findAll":
		matches := re.FindAllString(text, -1)
		if len(matches) == 0 {
			return "No matches found", nil
		}
		var buf bytes.Buffer
		for i, m := range matches {
			buf.WriteString(fmt.Sprintf("%d: %s\n", i+1, m))
		}
		return buf.String(), nil

	case "replace":
		replacement, _ := input["replacement"].(string)
		result := re.ReplaceAllString(text, replacement)
		return result, nil

	case "split":
		parts := re.Split(text, -1)
		return strings.Join(parts, "\n"), nil

	case "test":
		if re.MatchString(text) {
			return "true", nil
		}
		return "false", nil

	default:
		return "", fmt.Errorf("unsupported action: %s", action)
	}
}
