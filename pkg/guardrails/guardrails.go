// Package guardrails provides input/output validation middleware for agent execution.
// Guardrails intercept agent outputs to enforce quality, safety, and structural constraints.
package guardrails

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Ecook14/gocrewwai/pkg/llm"
)

// Guardrail defines the interface for all validation middleware.
// Implementations validate agent output and return an error if the output fails validation.
type Guardrail interface {
	// Name returns a human-readable identifier for this guardrail.
	Name() string
	// Validate checks the output string and returns nil if valid, or an error describing the failure.
	Validate(output string) error
}

// MaxTokenGuardrail rejects outputs exceeding a specified word count.
// Uses word-level approximation (1 word ≈ 1 token) for fast, dependency-free checking.
type MaxTokenGuardrail struct {
	MaxTokens int
}

func NewMaxTokenGuardrail(maxTokens int) *MaxTokenGuardrail {
	return &MaxTokenGuardrail{MaxTokens: maxTokens}
}

func (g *MaxTokenGuardrail) Name() string { return "MaxTokenGuardrail" }

func (g *MaxTokenGuardrail) Validate(output string) error {
	words := strings.Fields(output)
	if len(words) > g.MaxTokens {
		return fmt.Errorf("output has %d tokens, exceeds maximum of %d", len(words), g.MaxTokens)
	}
	return nil
}

// ContentFilterGuardrail blocks outputs containing any of the forbidden patterns.
// Patterns are compiled as regular expressions for flexible matching.
type ContentFilterGuardrail struct {
	ForbiddenPatterns []*regexp.Regexp
	RawPatterns       []string // stored for error messages
}

func NewContentFilterGuardrail(patterns []string) (*ContentFilterGuardrail, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern '%s': %w", p, err)
		}
		compiled = append(compiled, re)
	}
	return &ContentFilterGuardrail{
		ForbiddenPatterns: compiled,
		RawPatterns:       patterns,
	}, nil
}

func (g *ContentFilterGuardrail) Name() string { return "ContentFilterGuardrail" }

func (g *ContentFilterGuardrail) Validate(output string) error {
	for i, re := range g.ForbiddenPatterns {
		if re.MatchString(output) {
			return fmt.Errorf("output matches forbidden pattern '%s'", g.RawPatterns[i])
		}
	}
	return nil
}

// SchemaGuardrail validates that the output is valid JSON and can be unmarshalled
// into the provided schema template. The schema is a pointer to a Go struct.
type SchemaGuardrail struct {
	SchemaTemplate interface{}
}

func NewSchemaGuardrail(schema interface{}) *SchemaGuardrail {
	return &SchemaGuardrail{SchemaTemplate: schema}
}

func (g *SchemaGuardrail) Name() string { return "SchemaGuardrail" }

func (g *SchemaGuardrail) Validate(output string) error {
	trimmed := strings.TrimSpace(output)
	if !json.Valid([]byte(trimmed)) {
		return fmt.Errorf("output is not valid JSON")
	}
	// Attempt unmarshal to verify structural compatibility
	if err := json.Unmarshal([]byte(trimmed), g.SchemaTemplate); err != nil {
		return fmt.Errorf("output does not match expected schema: %w", err)
	}
	return nil
}

// RunAll executes all guardrails against the given output.
// Returns nil if all pass, or the first encountered error.
func RunAll(guardrails []Guardrail, output string) error {
	for _, g := range guardrails {
		if err := g.Validate(output); err != nil {
			return fmt.Errorf("guardrail '%s' failed: %w", g.Name(), err)
		}
	}
	return nil
}

// Elite Tier: Advanced Guardrails

// PIIRedactionGuardrail automatically redacts sensitive info (emails, SSNs).
type PIIRedactionGuardrail struct {
	EmailRegex *regexp.Regexp
	SSNRegex   *regexp.Regexp
}

func NewPIIRedactionGuardrail() *PIIRedactionGuardrail {
	return &PIIRedactionGuardrail{
		EmailRegex: regexp.MustCompile(`[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,4}`),
		SSNRegex:   regexp.MustCompile(`\d{3}-\d{2}-\d{4}`),
	}
}

func (g *PIIRedactionGuardrail) Name() string { return "PIIRedactionGuardrail" }

func (g *PIIRedactionGuardrail) Validate(output string) error {
	// In an "Elite" implementation, this might actually MODIFY the output.
	// But in a strict Guardrail interface, we reject if PII exists.
	if g.EmailRegex.MatchString(output) || g.SSNRegex.MatchString(output) {
		return fmt.Errorf("PII (Email or SSN) detected in output")
	}
	return nil
}

// ToxicityGuardrail filters for harmful content.
type ToxicityGuardrail struct {
	ToxicWords []string
}

func NewToxicityGuardrail() *ToxicityGuardrail {
	return &ToxicityGuardrail{
		ToxicWords: []string{"hate", "violence", "harmful"}, // Simplified
	}
}

func (g *ToxicityGuardrail) Name() string { return "ToxicityGuardrail" }

func (g *ToxicityGuardrail) Validate(output string) error {
	lower := strings.ToLower(output)
	for _, word := range g.ToxicWords {
		if strings.Contains(lower, word) {
			return fmt.Errorf("toxic content detected: word '%s' is forbidden", word)
		}
	}
	return nil
}

// LLMReviewGuardrail uses a secondary LLM call to validate the primary agent's output.
// This is the gold standard for "Elite" guardrailing.
type Reviewer interface {
	Generate(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (string, error)
}

type LLMReviewGuardrail struct {
	Reviewer Reviewer
	Criteria string
}

// Advanced Implementation: Uses a decoupled interface to prevent package circularity.
// This allows the Reviewer to be any implementation of a Generate method (e.g. pkg/llm).
func (g *LLMReviewGuardrail) Name() string { return "LLMReviewGuardrail" }

func (g *LLMReviewGuardrail) Validate(output string) error {
	ctx := context.Background()
	prompt := fmt.Sprintf("Review the following agent output based on these criteria: %s\n\nOutput: %s\n\nReturn 'PASS' if it satisfies the criteria, otherwise return a reason for failure.", g.Criteria, output)
	
	msgList := []llm.Message{
		{Role: "system", Content: "You are a strict output validator."},
		{Role: "user", Content: prompt},
	}

	review, err := g.Reviewer.Generate(ctx, msgList, llm.GenerateOptions{})
	if err != nil {
		return fmt.Errorf("llm review failed: %w", err)
	}

	if !strings.Contains(strings.ToUpper(review), "PASS") {
		return fmt.Errorf("llm review rejected output: %s", review)
	}
	return nil
}
