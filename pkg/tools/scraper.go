package tools

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ScraperTool reads the raw text content of a web page.
type ScraperTool struct {
	BaseTool
}

const (
	// maxScraperBodyBytes caps fetched HTML before text extraction.
	maxScraperBodyBytes = 1 << 20 // 1MB
	// maxScraperTextLen caps extracted text returned to the agent.
	maxScraperTextLen = 15 * 1024
)

func NewScraperTool() *ScraperTool {
	return &ScraperTool{
		BaseTool: BaseTool{
			NameValue:        "WebScraper",
			DescriptionValue: "Reads the text content of a URL. Useful for gathering detailed information from a specific webpage.",
		},
	}
}

func (t *ScraperTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	urlStr, ok := input["url"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'url' parameter")
	}

	// SSRF protection: validate the URL before fetching
	if err := t.validateURL(urlStr); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Gocrew Agent/1.0 (WebScraper)")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        5,
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     30 * time.Second,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("webpage returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxScraperBodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to read body: %w", err)
	}

	body := string(bodyBytes)

	// Basic HTML-to-Text cleanup
	body = stripHTMLText(body)

	if len(body) > maxScraperTextLen {
		body = body[:maxScraperTextLen] + "\n... [Content Truncated]"
	}

	return strings.TrimSpace(body), nil
}

func (t *ScraperTool) validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http and https schemes are allowed, got %s", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("url has no host")
	}

	// Block well-known cloud metadata endpoints.
	metadataHosts := []string{
		"169.254.169.254",
		"metadata.google.internal",
		"metadata.google",
		"metadata",
		"100.100.100.200",
	}
	for _, mh := range metadataHosts {
		if strings.EqualFold(host, mh) {
			slog.Warn("blocked cloud metadata endpoint access attempt", "host", host, "tool", "WebScraper")
			return fmt.Errorf("access to cloud metadata endpoint %s is blocked", host)
		}
	}

	// Block the metadata path prefix regardless of host.
	if strings.HasPrefix(u.Path, "/metadata") || strings.HasPrefix(u.Path, "/latest/meta-data") {
		return fmt.Errorf("access to metadata paths is blocked")
	}

	// Resolve the host to IP addresses and validate each one.
	addrs, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("failed to resolve host %s: %w", host, err)
	}

	if len(addrs) == 0 {
		return fmt.Errorf("no IP addresses resolved for %s", host)
	}

	for _, addr := range addrs {
		if addr == nil {
			continue
		}
		if t.isBlockedIP(addr) {
			return fmt.Errorf("target IP %s is in a blocked range", addr.String())
		}
	}

	return nil
}

func (t *ScraperTool) isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.IsMulticast() {
		return true
	}
	if ip.IsUnspecified() {
		return true
	}
	return false
}

func stripHTMLText(html string) string {
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

func (t *ScraperTool) Name() string        { return t.BaseTool.NameValue }
func (t *ScraperTool) Description() string { return t.BaseTool.DescriptionValue }
