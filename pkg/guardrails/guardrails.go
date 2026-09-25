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
type Guardrail interface {
	Name() string
	Validate(output string) error
}

// MaxTokenGuardrail rejects outputs exceeding a specified word count.
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

// ContentFilterGuardrail blocks outputs matching forbidden regex patterns.
type ContentFilterGuardrail struct {
	ForbiddenPatterns []*regexp.Regexp
	RawPatterns       []string
	CaseInsensitive   bool
	Severity          string // "error" (default), "warn", "silent"
}

func NewContentFilterGuardrail(patterns []string, opts ...ContentFilterOption) (*ContentFilterGuardrail, error) {
	c := &ContentFilterGuardrail{
		CaseInsensitive: false,
		Severity:        "error",
	}
	for _, opt := range opts {
		opt(c)
	}
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern '%s': %w", p, err)
		}
		if c.CaseInsensitive {
			re = regexp.MustCompile(`(?i)` + p)
		}
		compiled = append(compiled, re)
	}
	return &ContentFilterGuardrail{
		ForbiddenPatterns: compiled,
		RawPatterns:       patterns,
	}, nil
}

type ContentFilterOption func(*ContentFilterGuardrail)

func WithCaseInsensitive() ContentFilterOption {
	return func(c *ContentFilterGuardrail) { c.CaseInsensitive = true }
}

func WithSeverity(s string) ContentFilterOption {
	return func(c *ContentFilterGuardrail) { c.Severity = s }
}

func (g *ContentFilterGuardrail) Name() string { return "ContentFilterGuardrail" }

func (g *ContentFilterGuardrail) Validate(output string) error {
	for i, re := range g.ForbiddenPatterns {
		if re.MatchString(output) {
			if g.Severity == "warn" {
				return nil
			}
			return fmt.Errorf("output matches forbidden pattern '%s'", g.RawPatterns[i])
		}
	}
	return nil
}

// SchemaGuardrail validates output against a JSON schema template.
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
	if err := json.Unmarshal([]byte(trimmed), g.SchemaTemplate); err != nil {
		return fmt.Errorf("output does not match expected schema: %w", err)
	}
	return nil
}

// RunAll executes all guardrails and returns the first error.
// AllViolations returns every violation instead of stopping at the first failure.
func AllViolations(guardrails []Guardrail, output string) []string {
	var violations []string
	for _, g := range guardrails {
		if err := g.Validate(output); err != nil {
			violations = append(violations, fmt.Sprintf("[%s] %s", g.Name(), err.Error()))
		}
	}
	return violations
}

func RunAll(guardrails []Guardrail, output string) error {
	for _, g := range guardrails {
		if err := g.Validate(output); err != nil {
			return fmt.Errorf("guardrail '%s' failed: %w", g.Name(), err)
		}
	}
	return nil
}

// Elite Tier: Advanced Guardrails

// PIIRedactionGuardrail detects PII in outputs.
type PIIRedactionGuardrail struct {
	EmailRegex *regexp.Regexp
	SSNRegex   *regexp.Regexp
	PhoneRegex *regexp.Regexp
	ZipRegex   *regexp.Regexp
	Config     PIIRedactionConfig
}

// PIIRedactionConfig configures PII detection behavior.
type PIIRedactionConfig struct {
	Enabled         bool     // Enable/disable all PII detection
	AllowedPatterns []string // Exceptions: patterns that are always allowed
	RedactOutput    bool     // If true, redact PII from output instead of rejecting
	AllowedFields   []string // Field/key names where PII is allowed (JSON/key-value contexts)
}

func DefaultPIIConfig() PIIRedactionConfig {
	return PIIRedactionConfig{
		Enabled:      true,
		RedactOutput: false,
	}
}

func NewPIIRedactionGuardrail() *PIIRedactionGuardrail {
	cfg := DefaultPIIConfig()
	return &PIIRedactionGuardrail{
		EmailRegex: regexp.MustCompile(`[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,4}`),
		SSNRegex:   regexp.MustCompile(`\d{3}-\d{2}-\d{4}`),
		PhoneRegex: regexp.MustCompile(`(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}`),
		ZipRegex:   regexp.MustCompile(`\b\d{5}(?:-\d{4})?\b`),
		Config:     cfg,
	}
}

func WithPIIConfig(cfg PIIRedactionConfig) func(*PIIRedactionGuardrail) {
	return func(g *PIIRedactionGuardrail) { g.Config = cfg }
}

func (g *PIIRedactionGuardrail) Name() string { return "PIIRedactionGuardrail" }

func (g *PIIRedactionGuardrail) Validate(output string) error {
	if !g.Config.Enabled {
		return nil
	}
	hasPII := g.EmailRegex.MatchString(output) || g.SSNRegex.MatchString(output) ||
		g.PhoneRegex.MatchString(output) || g.ZipRegex.MatchString(output)
	if hasPII {
		if g.Config.RedactOutput {
			return nil // silently redact — caller handles redaction
		}
		return fmt.Errorf("PII detected in output")
	}
	return nil
}

// ToxicityGuardrail filters for harmful content.
type ToxicityGuardrail struct {
	ToxicWords        []string
	CaseInsensitive   bool
	WholeWordOnly     bool
	MaxDensity        float64 // max fraction of words that may be toxic (0 = unlimited)
	defaultToxicWords []string
}

var defaultToxicWordList = []string{
	"hate", "violence", "harmful", "discrimination", "harassment",
	"self-harm", "threat", "abuse", "racist", "sexist", "homophobic",
	"transphobic", "ableist", "extremist", "terrorist", "illegal",
}

func NewToxicityGuardrail(opts ...ToxicityOption) *ToxicityGuardrail {
	t := &ToxicityGuardrail{
		ToxicWords:        make([]string, len(defaultToxicWordList)),
		CaseInsensitive:   true,
		WholeWordOnly:     false,
		MaxDensity:        0,
		defaultToxicWords: defaultToxicWordList,
	}
	copy(t.ToxicWords, defaultToxicWordList)
	for _, opt := range opts {
		opt(t)
	}
	return t
}

type ToxicityOption func(*ToxicityGuardrail)

func WithToxicWords(words []string) ToxicityOption {
	return func(t *ToxicityGuardrail) {
		t.ToxicWords = make([]string, len(words))
		copy(t.ToxicWords, words)
	}
}

func WithToxicityCaseSensitive() ToxicityOption {
	return func(t *ToxicityGuardrail) { t.CaseInsensitive = false }
}

func WithToxicityWholeWord() ToxicityOption {
	return func(t *ToxicityGuardrail) { t.WholeWordOnly = true }
}

func WithToxicityMaxDensity(d float64) ToxicityOption {
	return func(t *ToxicityGuardrail) { t.MaxDensity = d }
}

func (g *ToxicityGuardrail) Name() string { return "ToxicityGuardrail" }

func (g *ToxicityGuardrail) Validate(output string) error {
	lower := output
	if g.CaseInsensitive {
		lower = strings.ToLower(output)
	}
	words := strings.Fields(lower)
	toxicCount := 0
	for _, word := range g.ToxicWords {
		target := word
		if g.CaseInsensitive {
			target = strings.ToLower(word)
		}
		if g.WholeWordOnly {
			for _, w := range words {
				if w == target {
					toxicCount++
				}
			}
		} else {
			if strings.Contains(lower, target) {
				toxicCount++
			}
		}
	}
	if g.MaxDensity > 0 && len(words) > 0 {
		density := float64(toxicCount) / float64(len(words))
		if density > g.MaxDensity {
			return fmt.Errorf("toxic content density %.2f exceeds max %.2f", density, g.MaxDensity)
		}
	}
	if toxicCount > 0 && g.MaxDensity == 0 {
		return fmt.Errorf("toxic content detected")
	}
	return nil
}

// Reviewer is the interface for LLM-based review (decoupled to prevent circularity).
type Reviewer interface {
	Generate(ctx context.Context, messages []llm.Message, options llm.GenerateOptions) (string, error)
}

// LLMReviewGuardrail uses a secondary LLM to validate outputs.
type LLMReviewGuardrail struct {
	Reviewer Reviewer
	Criteria string
}

func (g *LLMReviewGuardrail) Name() string { return "LLMReviewGuardrail" }

func (g *LLMReviewGuardrail) Validate(ctx context.Context, output string) error {
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

// ValidatorFunc is a function type for custom validators.
type ValidatorFunc func(output string) error

// ValidatorGuardrail wraps a ValidatorFunc as a Guardrail.
type ValidatorGuardrail struct {
	fn   ValidatorFunc
	name string
}

func NewValidatorGuardrail(name string, fn ValidatorFunc) *ValidatorGuardrail {
	return &ValidatorGuardrail{
		name: name,
		fn:   fn,
	}
}

func (g *ValidatorGuardrail) Name() string { return g.name }

func (g *ValidatorGuardrail) Validate(output string) error {
	return g.fn(output)
}

// NoEmptyString returns a validator that rejects empty output.
func NoEmptyString() ValidatorFunc {
	return func(output string) error {
		if strings.TrimSpace(output) == "" {
			return fmt.Errorf("output is empty")
		}
		return nil
	}
}

// MinLength returns a validator that rejects output shorter than n characters.
func MinLength(n int) ValidatorFunc {
	return func(output string) error {
		if len(output) < n {
			return fmt.Errorf("output too short: %d < %d", len(output), n)
		}
		return nil
	}
}

// MaxLength returns a validator that rejects output longer than n characters.
func MaxLength(n int) ValidatorFunc {
	return func(output string) error {
		if len(output) > n {
			return fmt.Errorf("output too long: %d > %d", len(output), n)
		}
		return nil
	}
}
