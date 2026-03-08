package guardrails

import (
	"fmt"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// Input Sanitization — Security Middleware for Agent Inputs
// ---------------------------------------------------------------------------

// Sanitizer provides configurable input sanitization for agent prompts and tool inputs.
type Sanitizer struct {
	// MaxInputLength is the maximum allowed character count (0 = unlimited)
	MaxInputLength int
	// BlockedPatterns are regex patterns that will cause rejection
	BlockedPatterns []*regexp.Regexp
	// StripPatterns are regex patterns that will be removed from input
	StripPatterns []*regexp.Regexp
	// AllowHTML controls whether HTML tags are permitted
	AllowHTML bool
}

// DefaultSanitizer creates a sanitizer with sensible security defaults.
func DefaultSanitizer() *Sanitizer {
	return &Sanitizer{
		MaxInputLength: 50000, // 50KB
		BlockedPatterns: []*regexp.Regexp{
			// Block common prompt injection patterns
			regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|prior|above)\s+(instructions?|rules?|prompts?)`),
			regexp.MustCompile(`(?i)you\s+are\s+now\s+in\s+(developer|admin|debug|unrestricted)\s+mode`),
			regexp.MustCompile(`(?i)system\s*:\s*you\s+are`),
			regexp.MustCompile(`(?i)forget\s+(everything|all)\s+(you|that|i)`),
			regexp.MustCompile(`(?i)new\s+instructions?\s*:`),
			regexp.MustCompile(`(?i)override\s+(system|safety|security)`),
		},
		StripPatterns: []*regexp.Regexp{
			// Strip null bytes and control characters (except newline/tab)
			regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`),
		},
		AllowHTML: false,
	}
}

// SanitizeResult holds the outcome of a sanitization check.
type SanitizeResult struct {
	Clean   bool   // Whether the input passed all checks
	Output  string // The sanitized input (or original if rejected)
	Reason  string // Why it was rejected (empty if clean)
}

// Sanitize processes input through all configured checks.
func (s *Sanitizer) Sanitize(input string) SanitizeResult {
	// Length check
	if s.MaxInputLength > 0 && len(input) > s.MaxInputLength {
		return SanitizeResult{
			Clean:  false,
			Output: input,
			Reason: fmt.Sprintf("input exceeds maximum length (%d > %d)", len(input), s.MaxInputLength),
		}
	}

	// Strip patterns
	cleaned := input
	for _, pattern := range s.StripPatterns {
		cleaned = pattern.ReplaceAllString(cleaned, "")
	}

	// Strip HTML if not allowed
	if !s.AllowHTML {
		cleaned = stripHTMLTags(cleaned)
	}

	// Check blocked patterns
	for _, pattern := range s.BlockedPatterns {
		if pattern.MatchString(cleaned) {
			return SanitizeResult{
				Clean:  false,
				Output: cleaned,
				Reason: fmt.Sprintf("input matches blocked pattern: %s", pattern.String()),
			}
		}
	}

	return SanitizeResult{
		Clean:  true,
		Output: cleaned,
	}
}

// AddBlockedPattern adds a custom blocked regex pattern.
func (s *Sanitizer) AddBlockedPattern(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	s.BlockedPatterns = append(s.BlockedPatterns, re)
	return nil
}

// AddStripPattern adds a custom strip regex pattern.
func (s *Sanitizer) AddStripPattern(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	s.StripPatterns = append(s.StripPatterns, re)
	return nil
}

// stripHTMLTags removes HTML tags from text.
func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// SanitizeMap sanitizes all string values in a map (for tool inputs).
func (s *Sanitizer) SanitizeMap(input map[string]interface{}) (map[string]interface{}, *SanitizeResult) {
	result := make(map[string]interface{}, len(input))
	for k, v := range input {
		str, ok := v.(string)
		if !ok {
			result[k] = v
			continue
		}
		sr := s.Sanitize(str)
		if !sr.Clean {
			return nil, &sr
		}
		result[k] = sr.Output
	}
	return result, nil
}
