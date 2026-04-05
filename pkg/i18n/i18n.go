package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"
	"strings"
)

//go:embed translations/*.json
var translationsFS embed.FS

// I18N handles loading and retrieving internationalized prompts.
type I18N struct {
	mu      sync.RWMutex
	prompts map[string]map[string]interface{}
	lang    string
}

// NewI18N creates a new I18N instance for a given language.
func NewI18N(lang string) (*I18N, error) {
	if lang == "" {
		lang = "en"
	}

	i := &I18N{lang: lang}
	if err := i.loadPrompts(); err != nil {
		return nil, err
	}
	return i, nil
}

func (i *I18N) loadPrompts() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	fileName := fmt.Sprintf("translations/%s.json", i.lang)
	data, err := translationsFS.ReadFile(fileName)
	if err != nil {
		// Fallback to English if language not found
		if i.lang != "en" {
			data, err = translationsFS.ReadFile("translations/en.json")
		}
		if err != nil {
			return fmt.Errorf("i18n: failed to load translations: %w", err)
		}
	}

	if err := json.Unmarshal(data, &i.prompts); err != nil {
		return fmt.Errorf("i18n: failed to parse translations: %w", err)
	}

	return nil
}

// Retrieve returns a prompt by category and key.
func (i *I18N) Retrieve(kind, key string) string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if category, ok := i.prompts[kind]; ok {
		if val, ok := category[key]; ok {
			if s, ok := val.(string); ok {
				return s
			}
			// Handle nested objects (like 'add_image' in tools)
			if m, ok := val.(map[string]interface{}); ok {
				if name, ok := m["name"].(string); ok {
					return name
				}
			}
		}
	}
	return fmt.Sprintf("[%s:%s not found]", kind, key)
}

// Process replaces placeholders in the form of {key} with values from the map.
func (i *I18N) Process(template string, vars map[string]string) string {
	result := template
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result
}

// Slice returns a prompt from the 'slices' category.
func (i *I18N) Slice(key string) string {
	return i.Retrieve("slices", key)
}

// Error returns a prompt from the 'errors' category.
func (i *I18N) Error(key string) string {
	return i.Retrieve("errors", key)
}

// Tool returns a prompt from the 'tools' category.
func (i *I18N) Tool(key string) string {
	return i.Retrieve("tools", key)
}

// Memory returns a prompt from the 'memory' category.
func (i *I18N) Memory(key string) string {
	return i.Retrieve("memory", key)
}
