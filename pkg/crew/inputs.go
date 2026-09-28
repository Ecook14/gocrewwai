package crew

import (
	"context"
	"fmt"
	"strings"
)

// MaxInputValueBytes bounds a single input value (prompt-injection/DoS guard).
const MaxInputValueBytes = 20000

// InterpolateInputs renders {{var}} placeholders in template from inputs.
// Every placeholder must have a matching key; unknown or empty keys are an
// error (fail closed — no silent empty substitution). Values are bounded.
func InterpolateInputs(template string, inputs map[string]string) (string, error) {
	var missing []string
	result := template
	// Collect distinct placeholder names.
	seen := make(map[string]struct{})
	for {
		start := strings.Index(result, "{{")
		if start < 0 {
			break
		}
		end := strings.Index(result[start+2:], "}}")
		if end < 0 {
			return "", fmt.Errorf("crew: unterminated placeholder in %q", template)
		}
		name := strings.TrimSpace(result[start+2 : start+2+end])
		if name == "" {
			return "", fmt.Errorf("crew: empty placeholder in %q", template)
		}
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			val, ok := inputs[name]
			if !ok || val == "" {
				missing = append(missing, name)
			}
		}
		// Replace one occurrence to keep scanning simple and bounded.
		val := inputs[name]
		if len(val) > MaxInputValueBytes {
			return "", fmt.Errorf("crew: input %q exceeds %d bytes", name, MaxInputValueBytes)
		}
		result = result[:start] + val + result[start+2+end+2:]
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("crew: missing inputs for placeholders: %s", strings.Join(missing, ", "))
	}
	return result, nil
}

// applyInputs interpolates task Description/ExpectedOutput in place.
func (c *Crew) applyInputs(inputs map[string]string) error {
	if inputs == nil {
		inputs = map[string]string{}
	}
	for _, t := range c.Tasks {
		if t == nil {
			continue
		}
		if !strings.Contains(t.Description, "{{") && !strings.Contains(t.ExpectedOutput, "{{") {
			continue
		}
		rendered, err := InterpolateInputs(t.Description, inputs)
		if err != nil {
			return fmt.Errorf("crew: task %q description: %w", t.Description, err)
		}
		t.Description = rendered
		if t.ExpectedOutput != "" {
			rendered, err := InterpolateInputs(t.ExpectedOutput, inputs)
			if err != nil {
				return fmt.Errorf("crew: task expected_output: %w", err)
			}
			t.ExpectedOutput = rendered
		}
	}
	return nil
}

// KickoffWithInputs starts execution after rendering {{var}} placeholders in
// task descriptions from inputs (CrewAI-compatible `kickoff(inputs={...})`).
func (c *Crew) KickoffWithInputs(ctx context.Context, inputs map[string]string) (interface{}, error) {
	if err := c.applyInputs(inputs); err != nil {
		return nil, err
	}
	return c.Kickoff(ctx)
}
