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

// ScrapeWebsiteTool fetches text content from a simple HTTP URL.
// Requires human review for arbitrary URL fetching (SSRF vector).
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

	// SSRF protection: validate the URL before fetching
	if err := t.validateURL(urlStr); err != nil {
		return "", err
	}

	if t.Options != nil && t.Options["verbose"] == true {
		slog.Info("Tool [Scrape Website]: Scraping URL: " + urlStr)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

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
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("failed with status code %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return "", fmt.Errorf("failed to read body: %w", err)
	}

	body := string(bodyBytes)
	if len(body) > 10000 {
		body = body[:10000] + "\n... [Output Truncated]"
	}

	return strings.TrimSpace(body), nil
}

func (t *ScrapeWebsiteTool) validateURL(rawURL string) error {
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
			slog.Warn("blocked cloud metadata endpoint access attempt", "host", host, "tool", "ScrapeWebsiteTool")
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

func (t *ScrapeWebsiteTool) isBlockedIP(ip net.IP) bool {
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

func (t *ScrapeWebsiteTool) RequiresReview() bool { return true }
func (t *ScrapeWebsiteTool) Name() string         { return t.BaseTool.NameValue }
func (t *ScrapeWebsiteTool) Description() string  { return t.BaseTool.DescriptionValue }

var _ Tool = (*ScrapeWebsiteTool)(nil)
