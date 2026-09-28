package guardrails

import (
	"testing"
)

func TestXMLValidator(t *testing.T) {
	g := NewXMLValidator("result")
	if err := g.Validate(`<result><a>1</a></result>`); err != nil {
		t.Errorf("valid XML rejected: %v", err)
	}
	if err := g.Validate(`<other/>`); err == nil {
		t.Error("wrong root accepted")
	}
	if err := g.Validate(`<unclosed>`); err == nil {
		t.Error("malformed XML accepted")
	}
	if txt, err := ExtractXMLText(`<r><v>hello</v></r>`, "v"); err != nil || txt != "hello" {
		t.Errorf("extract = %q, %v", txt, err)
	}
}

func TestCSVValidator(t *testing.T) {
	g := NewCSVValidator(2)
	if err := g.Validate("a,b\n1,2\n"); err != nil {
		t.Errorf("valid CSV rejected: %v", err)
	}
	if err := g.Validate("a,b,c\n1,2\n"); err == nil {
		t.Error("ragged CSV accepted")
	}
	rows, err := ParseCSV("a,b\n1,2\n", 2)
	if err != nil || len(rows) != 2 || rows[1][0] != "1" {
		t.Errorf("parse = %v, %v", rows, err)
	}
}

func TestRegexValidator(t *testing.T) {
	g, err := NewRegexValidator(`^Final Answer:`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if err := g.Validate("Final Answer: done"); err != nil {
		t.Errorf("match rejected: %v", err)
	}
	if err := g.Validate("nope"); err == nil {
		t.Error("non-match accepted")
	}
	if _, _, err := ExtractRegex("id=42", `id=(\d+)`); err != nil {
		t.Errorf("extract: %v", err)
	}
	if _, err := NewRegexValidator(`(`); err == nil {
		t.Error("bad pattern accepted")
	}
}
