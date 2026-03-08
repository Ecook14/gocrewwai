package llm

import (
	"os"
	"testing"
)

func TestFileCache(t *testing.T) {
	tempDir := "./test_cache"
	defer os.RemoveAll(tempDir)

	cache := NewFileCache(tempDir)

	key := "test_key"
	value := "test_value"

	// Test Set
	err := cache.Set(key, value)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Test Get
	got, ok := cache.Get(key)
	if !ok {
		t.Fatal("Failed to get cache")
	}
	if got != value {
		t.Errorf("Expected %s, got %s", value, got)
	}

	// Test Get non-existent
	_, ok = cache.Get("non_existent")
	if ok {
		t.Fatal("Expected false for non-existent key")
	}
}

func TestGenerateCacheKey(t *testing.T) {
	model := "gpt-4"
	prompt := "hello"
	options := map[string]interface{}{"temp": 0.7}

	key1 := GenerateCacheKey(model, prompt, options)
	key2 := GenerateCacheKey(model, prompt, options)

	if key1 != key2 {
		t.Fatal("Keys should be stable for same input")
	}

	options2 := map[string]interface{}{"temp": 0.8}
	key3 := GenerateCacheKey(model, prompt, options2)
	if key1 == key3 {
		t.Fatal("Keys should differ for different options")
	}
}
