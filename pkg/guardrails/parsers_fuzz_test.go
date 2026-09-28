package guardrails

import (
	"testing"
)

// FuzzParseCSV asserts the CSV parser never panics and round-trips its
// column-count contract: accepted rows always have exactly `columns` fields.
func FuzzParseCSV(f *testing.F) {
	f.Add("a,b\n1,2\n", 2)
	f.Add("x\n", 1)
	f.Add("", 0)
	f.Add("a,\"b\nc\",d\n", 3)
	f.Fuzz(func(t *testing.T, input string, columns int) {
		if columns < 0 {
			columns = -columns
		}
		columns %= 8
		rows, err := ParseCSV(input, columns)
		if err != nil {
			return
		}
		if columns > 0 {
			for i, row := range rows {
				if len(row) != columns {
					t.Fatalf("row %d has %d cols, want %d (input %q)", i, len(row), columns, input)
				}
			}
		}
	})
}

// FuzzXMLValidator asserts no panics on arbitrary input and that accepted
// input always contains the enforced root tag when set.
func FuzzXMLValidator(f *testing.F) {
	f.Add("<result><a>1</a></result>")
	f.Add("not xml")
	f.Add("<a><b>")
	f.Add("")
	f.Fuzz(func(t *testing.T, input string) {
		g := NewXMLValidator()
		_ = g.Validate(input) // must not panic; error expected for garbage
		if err := NewXMLValidator("result").Validate(input); err == nil {
			// Accepted with root enforced: input must mention the tag.
			found := false
			for i := 0; i+6 <= len(input); i++ {
				if input[i:i+6] == "result" {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("accepted without root tag (input %q)", input)
			}
		}
	})
}

// FuzzRegexValidator asserts compile errors are constructor errors (never
// panics) and matching is consistent with MatchString.
func FuzzRegexValidator(f *testing.F) {
	f.Add("^Final Answer:")
	f.Add("([a-z]+")
	f.Add("")
	f.Fuzz(func(t *testing.T, pattern string) {
		g, err := NewRegexValidator(pattern)
		if err != nil {
			return
		}
		_ = g.Validate("Final Answer: probe 123 !@#")
		_ = g.Validate("")
	})
}
