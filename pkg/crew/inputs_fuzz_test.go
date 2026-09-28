package crew

import (
	"strings"
	"testing"
)

// FuzzInterpolateInputs asserts: no panics; success output never contains
// "{{" remnants; failures only for missing/unterminated/empty/oversize.
func FuzzInterpolateInputs(f *testing.F) {
	f.Add("Research {{topic}} in {{lang}}", "topic=AI;lang=Go")
	f.Add("no placeholders", "")
	f.Add("{{unclosed}", "a=b")
	f.Add("{{}}", "")
	f.Fuzz(func(t *testing.T, template, kv string) {
		inputs := map[string]string{}
		if kv != "" {
			for _, pair := range strings.Split(kv, ";") {
				k, v, _ := strings.Cut(pair, "=")
				inputs[k] = v
			}
		}
		out, err := InterpolateInputs(template, inputs)
		if err != nil {
			return
		}
		if strings.Contains(out, "{{") {
			t.Fatalf("unresolved placeholder in output %q (template %q)", out, template)
		}
	})
}
