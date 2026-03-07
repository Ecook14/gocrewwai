package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ScrapeWebsiteTool fetches text content from a simple HTTP URL.
type ScrapeWebsiteTool struct {
	Options map[string]interface{}
}

func NewScrapeWebsiteTool() *ScrapeWebsiteTool {
	return &ScrapeWebsiteTool{}
}

func (t *ScrapeWebsiteTool) Name() string {
	return "ScrapeWebsiteTool"
}

func (t *ScrapeWebsiteTool) Description() string {
	return "Scrapes text content from a provided URL. Input requires 'url' as a string."
}

func (t *ScrapeWebsiteTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	urlRaw, ok := input["url"]
	if !ok {
		return "", fmt.Errorf("missing 'url' in input")
	}

	urlStr, ok := urlRaw.(string)
	if !ok {
		return "", fmt.Errorf("'url' must be a string")
	}

	// Simple HTTP GET request mapping
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("failed with status code %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body: %w", err)
	}

	// Just return raw body text (simulating basic scraping string extraction)
	body := string(bodyBytes)
	
	// Optional trim to prevent massive context dumps
	if len(body) > 10000 {
		body = body[:10000] + "\n... [Output Truncated]"
	}

	return strings.TrimSpace(body), nil
}
