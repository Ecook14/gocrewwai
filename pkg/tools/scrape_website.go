package tools

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// ScrapeWebsiteTool fetches text content from a simple HTTP URL.
type ScrapeWebsiteTool struct {
	BaseTool
	Options map[string]interface{}
}

func NewScrapeWebsiteTool() *ScrapeWebsiteTool {
	return &ScrapeWebsiteTool{
		BaseTool: BaseTool{
			NameValue:        "ScrapeWebsiteTool",
			DescriptionValue: "Scrapes text content from a provided URL. Input requires 'url' as a string.",
		},
	}
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

	if t.Options != nil && t.Options["verbose"] == true {
		slog.Info("Tool [Scrape Website]: Scraping URL", slog.String("url", urlStr))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
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

	body := string(bodyBytes)
	if len(body) > 10000 {
		body = body[:10000] + "\n... [Output Truncated]"
	}

	return strings.TrimSpace(body), nil
}

func (t *ScrapeWebsiteTool) RequiresReview() bool { return false }
