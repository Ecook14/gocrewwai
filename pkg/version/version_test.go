package version

import "testing"

func TestDefaultIsDevNotAStaleRelease(t *testing.T) {
	// A local `go test` / `go run` build must never claim to be a release.
	if Version == "v0.9.0" {
		t.Fatalf("Version hardcoded to a release number (%q); it must default to dev "+
			"and be set with -ldflags -X at build time", Version)
	}
}

func TestString(t *testing.T) {
	oldV, oldC := Version, Commit
	t.Cleanup(func() { Version, Commit = oldV, oldC })

	// String delegates to Display, so a "v"-prefixed tag renders without it.
	Version, Commit = "v1.0.0", "unknown"
	if got := String(); got != "1.0.0" {
		t.Fatalf("String() = %q, want 1.0.0", got)
	}

	Version, Commit = "v1.0.0", "0123456789abcdef"
	if got := String(); got != "1.0.0 (0123456)" {
		t.Fatalf("String() = %q, want %q", got, "1.0.0 (0123456)")
	}
}

func TestDisplayStripsVAndKeepsDev(t *testing.T) {
	old := Version
	t.Cleanup(func() { Version = old })

	for in, want := range map[string]string{
		"v1.0.0-beta.3": "1.0.0-beta.3",
		"1.0.0":         "1.0.0",
		"dev":           "dev",
		"":              "dev",
	} {
		Version = in
		if got := Display(); got != want {
			t.Errorf("Display() with Version=%q = %q, want %q", in, got, want)
		}
	}
}

func TestShortCommit(t *testing.T) {
	cases := []struct{ in, want string }{
		{"abc", "abc"},
		{"0123456789ab", "0123456"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := shortCommitOf(tc.in); got != tc.want {
			t.Errorf("shortCommitOf(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
