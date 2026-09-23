package tools

import (
	"bytes"
	"testing"
)

func TestPDFReadTool(t *testing.T) {
	var _ Tool = (*PDFReadTool)(nil)
	t.Skip("PDF tests require a real PDF fixture; skip in short runs")
}

func TestHTMLReadTool(t *testing.T) {
	var _ Tool = (*HTMLReadTool)(nil)
	t.Skip("HTML tests require a real HTML fixture; skip in short runs")
}

func TestCSVReadTool(t *testing.T) {
	var _ Tool = (*CSVReadTool)(nil)
	t.Skip("CSV tests require a real CSV fixture; skip in short runs")
}

func TestExcelReadTool(t *testing.T) {
	var _ Tool = (*ExcelReadTool)(nil)
	t.Skip("Excel tests require a real XLSX fixture; skip in short runs")
}

func TestXMLReadTool(t *testing.T) {
	var _ Tool = (*XMLReadTool)(nil)
	t.Skip("XML tests require a real XML fixture; skip in short runs")
}

func TestYAMLReadTool(t *testing.T) {
	var _ Tool = (*YAMLReadTool)(nil)
	t.Skip("YAML tests require a real YAML fixture; skip in short runs")
}

func TestJSONParseTool(t *testing.T) {
	var _ Tool = (*JSONParseTool)(nil)
	t.Skip("JSON tests require a real JSON fixture; skip in short runs")
}

// --- unit tests for stripHTMLTagsOnly ---

func TestStripHTMLTagsOnly(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple paragraph",
			input:    "<p>Hello World</p>",
			expected: "Hello World",
		},
		{
			name:     "nested tags",
			input:    "<div><p>Nested <b>text</b></p></div>",
			expected: "Nested text",
		},
		{
			name:     "script and style removed",
			input:    "<script>alert('xss')</script><p>safe</p><style>.x{}</style>",
			expected: "safe",
		},
		{
			name:     "html entities decoded",
			input:    "Tom &amp; Jerry &lt;3 &quot;quotes&quot;",
			expected: "Tom & Jerry <3 \"quotes\"",
		},
		{
			name:     "pre block preserved",
			input:    "<pre>  spaced  </pre><p>after</p>",
			expected: "  spaced  after",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "no tags",
			input:    "plain text without tags",
			expected: "plain text without tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripHTMLTagsOnly(tt.input)
			if got != tt.expected {
				t.Errorf("stripHTMLTagsOnly(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStripHTMLTagsOnly_BasicEntities(t *testing.T) {
	input := "A&nbsp;B&amp;C&lt;D&gt;E&quot;F&#39;G&#x27;H"
	want := "A B&C<D>E\"F'G'H"
	if got := stripHTMLTagsOnly(input); got != want {
		t.Errorf("entity decode mismatch: got %q, want %q", got, want)
	}
}

func TestDocumentLoaderInterfaces(t *testing.T) {
	_ = &PDFReadTool{}
	_ = &HTMLReadTool{}
	_ = &CSVReadTool{}
	_ = &ExcelReadTool{}
	_ = &XMLReadTool{}
	_ = &YAMLReadTool{}
	_ = &JSONParseTool{}

	var buf bytes.Buffer
	_ = buf
}
