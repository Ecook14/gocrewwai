package memory

import (
	"testing"
)

func TestInScopeSegmentBoundary(t *testing.T) {
	cases := []struct {
		record, filter string
		want           bool
	}{
		{"/tenant/a", "/tenant/a", true},
		{"/tenant/a/project", "/tenant/a", true},
		{"/tenant/ab", "/tenant/a", false},
		{"/tenant/a", "/tenant/ab", false},
		{"/anything", "/", true},
		{"tenant/a", "/tenant/a", true},   // normalization
		{"/tenant/a/", "/tenant/a", true}, // trailing slash
	}
	for _, c := range cases {
		if got := inScope(c.record, c.filter); got != c.want {
			t.Errorf("inScope(%q, %q) = %v, want %v", c.record, c.filter, got, c.want)
		}
	}
}

func TestCopyMetadataDetaches(t *testing.T) {
	inner := map[string]interface{}{"label": "x"}
	meta := map[string]interface{}{"scope": "/a", "nested": inner, "tags": []interface{}{"t"}}
	out := copyMetadata(meta)
	out["scope"] = "/b"
	out["nested"].(map[string]interface{})["label"] = "y"
	out["tags"].([]interface{})[0] = "z"
	if meta["scope"] != "/a" {
		t.Error("outer map shared with store")
	}
	if inner["label"] != "x" {
		t.Error("nested map shared with store")
	}
	if meta["tags"].([]interface{})[0] != "t" {
		t.Error("slice shared with store")
	}
	if copyMetadata(nil) != nil {
		t.Error("copyMetadata(nil) should be nil")
	}
}
