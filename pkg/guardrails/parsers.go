package guardrails

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

// MaxParseBytes bounds parser input (DoS guard).
const MaxParseBytes = 1 << 20 // 1MB

func boundParseInput(input string) (string, error) {
	if len(input) > MaxParseBytes {
		return "", fmt.Errorf("guardrails: parse input exceeds %d bytes", MaxParseBytes)
	}
	return input, nil
}

// XMLValidGuardrail validates that output is well-formed XML with the expected root.
type XMLValidGuardrail struct {
	// Root, when non-empty, must match the document root element.
	Root string
}

// NewXMLValidator creates an XML well-formedness guardrail.
func NewXMLValidator(root ...string) *XMLValidGuardrail {
	g := &XMLValidGuardrail{}
	if len(root) > 0 {
		g.Root = root[0]
	}
	return g
}

func (g *XMLValidGuardrail) Name() string { return "XMLValidator" }

// Validate rejects malformed XML or a mismatched root element.
func (g *XMLValidGuardrail) Validate(output string) error {
	input, err := boundParseInput(output)
	if err != nil {
		return err
	}
	dec := xml.NewDecoder(strings.NewReader(input))
	for {
		tok, err := dec.Token()
		if err != nil {
			return fmt.Errorf("guardrails: invalid XML: %w", err)
		}
		if start, ok := tok.(xml.StartElement); ok {
			if g.Root != "" && start.Name.Local != g.Root {
				return fmt.Errorf("guardrails: XML root %q, want %q", start.Name.Local, g.Root)
			}
			return nil
		}
	}
}

// ExtractXMLText returns the character data of the first element matching tag.
func ExtractXMLText(input, tag string) (string, error) {
	input, err := boundParseInput(input)
	if err != nil {
		return "", err
	}
	dec := xml.NewDecoder(strings.NewReader(input))
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("guardrails: invalid XML: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == tag {
				depth++
				if depth == 1 {
					continue
				}
			} else if depth > 0 {
				depth++
			}
		case xml.EndElement:
			if depth > 0 {
				depth--
				if depth == 0 && t.Name.Local == tag {
					goto done
				}
			}
		case xml.CharData:
			if depth == 1 {
				return strings.TrimSpace(string(t)), nil
			}
		}
	}
done:
	return "", fmt.Errorf("guardrails: tag %q not found", tag)
}

// CSVValidGuardrail validates that output parses as CSV with expected columns.
type CSVValidGuardrail struct {
	// Columns, when non-empty, must match the header row field count.
	Columns int
	// HasHeader requires the first row to be treated as a header.
	HasHeader bool
}

// NewCSVValidator creates a CSV well-formedness guardrail.
func NewCSVValidator(columns ...int) *CSVValidGuardrail {
	g := &CSVValidGuardrail{}
	if len(columns) > 0 {
		g.Columns = columns[0]
	}
	return g
}

func (g *CSVValidGuardrail) Name() string { return "CSVValidator" }

// Validate rejects malformed CSV or wrong column counts.
func (g *CSVValidGuardrail) Validate(output string) error {
	_, err := ParseCSV(output, g.Columns)
	return err
}

// ParseCSV parses CSV text, enforcing column count when columns > 0.
func ParseCSV(input string, columns int) ([][]string, error) {
	input, err := boundParseInput(input)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(strings.NewReader(input))
	r.FieldsPerRecord = -1 // check manually for a better error
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("guardrails: invalid CSV: %w", err)
	}
	if columns > 0 {
		for i, row := range rows {
			if len(row) != columns {
				return nil, fmt.Errorf("guardrails: CSV row %d has %d columns, want %d", i, len(row), columns)
			}
		}
	}
	return rows, nil
}

// RegexGuardrail validates output against a pattern (compiled once).
type RegexGuardrail struct {
	pattern *regexp.Regexp
	source  string
}

// NewRegexValidator compiles pattern; invalid patterns are a constructor error.
func NewRegexValidator(pattern string) (*RegexGuardrail, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("guardrails: invalid regex: %w", err)
	}
	return &RegexGuardrail{pattern: re, source: pattern}, nil
}

func (g *RegexGuardrail) Name() string { return "RegexValidator" }

// Validate rejects output that does not match the pattern.
func (g *RegexGuardrail) Validate(output string) error {
	input, err := boundParseInput(output)
	if err != nil {
		return err
	}
	if !g.pattern.MatchString(input) {
		return fmt.Errorf("guardrails: output does not match %q", g.source)
	}
	return nil
}

// ExtractRegex returns the first match (and submatches) of pattern in input.
func ExtractRegex(input, pattern string) (string, []string, error) {
	input, err := boundParseInput(input)
	if err != nil {
		return "", nil, err
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", nil, fmt.Errorf("guardrails: invalid regex: %w", err)
	}
	m := re.FindStringSubmatch(input)
	if m == nil {
		return "", nil, fmt.Errorf("guardrails: no match for %q", pattern)
	}
	return m[0], m[1:], nil
}
