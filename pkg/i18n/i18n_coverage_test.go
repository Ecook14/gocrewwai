package i18n_test

import (
	"strings"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/i18n"
)

// TestI18N_SupportedLanguages_Parity ensures every supported language has
// full key parity with English (no silent fallback gaps).
func TestI18N_SupportedLanguages_Parity(t *testing.T) {
	en, err := i18n.NewI18N("en")
	if err != nil {
		t.Fatalf("failed to create en i18n: %v", err)
	}
	for _, lang := range i18n.SupportedLanguages() {
		if lang == "en" {
			continue
		}
		other, err := i18n.NewI18N(lang)
		if err != nil {
			t.Fatalf("failed to create %s i18n: %v", lang, err)
		}
		// Full key-set parity: every en kind:key must resolve in the other lang.
		for _, kind := range en.Kinds() {
			for _, key := range en.Keys(kind) {
				if !other.Has(kind, key) {
					t.Errorf("lang %s missing key %s:%s", lang, kind, key)
				}
			}
		}
		// Spot-check values are non-empty translated strings.
		for _, tc := range [][2]string{
			{"slices", "observation"},
			{"slices", "task"},
			{"slices", "role_playing"},
			{"slices", "tools"},
			{"slices", "format"},
			{"errors", "tool_usage_error"},
			{"errors", "wrong_tool_name"},
			{"tools", "delegate_work"},
			{"tools", "ask_question"},
			{"memory", "query_user"},
			{"reasoning", "initial_plan"},
			{"hierarchical_manager_agent", "role"},
		} {
			got := other.Retrieve(tc[0], tc[1])
			if strings.Contains(got, "not found") {
				t.Errorf("lang %s missing key %s:%s", lang, tc[0], tc[1])
			}
			if got == "" {
				t.Errorf("lang %s empty value for %s:%s", lang, tc[0], tc[1])
			}
		}
	}
}

// TestI18N_InvalidLang ensures language tags are validated (embed path safety).
func TestI18N_InvalidLang(t *testing.T) {
	for _, bad := range []string{"../en", "en/../secret", "EN;rm", "e", "english-language-name-too-long"} {
		if _, err := i18n.NewI18N(bad); err == nil {
			t.Errorf("expected error for language %q", bad)
		}
	}
}
