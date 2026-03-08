package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ScraperTool reads the raw text content of a web page.
type ScraperTool struct{}

func NewScraperTool() *ScraperTool {
	return &ScraperTool{}
}

func (t *ScraperTool) Name() string { return "WebScraper" }

func (t *ScraperTool) Description() string {
	return "Reads the text content of a URL. Useful for gathering detailed information from a specific webpage."
}

func (t *ScraperTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	urlStr, ok := input["url"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'url' parameter")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Crew-GO Agent/1.0 (WebScraper)")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("webpage returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body: %w", err)
	}

	body := string(bodyBytes)
	
	// Basic HTML-to-Text cleanup
	body = stripHTML(body)

	if len(body) > 15000 {
		body = body[:15000] + "\n... [Content Truncated]"
	}

	return strings.TrimSpace(body), nil
}

func stripHTML(html string) string {
	var sb strings.Builder
	inTag := false
	for _, r := range html {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func (t *ScraperTool) RequiresReview() bool { return false }
