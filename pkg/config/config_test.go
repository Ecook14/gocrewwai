package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_WithFile(t *testing.T) {
	// Reset the singleton for this test
	resetGetForTest()
	
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")
	configContent := `{
		"llm": {"provider": "openai", "model": "gpt-4o"},
		"providers": {"openai": {"api_key": "test"}},
		"models": {"gpt-4o": {"name": "GPT-4o"}}
	}`
	os.WriteFile(configPath, []byte(configContent), 0644)
	os.Setenv("CREW_CONFIG_PATH", configPath)
	defer os.Unsetenv("CREW_CONFIG_PATH")
	
	cfg := Get()
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
	if len(cfg.Models) == 0 {
		t.Error("Expected models to be loaded from config file")
	}
	if len(cfg.Providers) == 0 {
		t.Error("Expected providers to be loaded from config file")
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Get()
	if cfg == nil {
		t.Fatal("Get() should return non-nil config")
	}
	// Even without config.json, maps should be initialized
	if cfg.Providers == nil {
		t.Error("Expected providers map to be initialized")
	}
	if cfg.Models == nil {
		t.Error("Expected models map to be initialized")
	}
	if cfg.Tools == nil {
		t.Error("Expected tools map to be initialized")
	}
}
