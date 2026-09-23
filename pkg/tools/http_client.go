package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPTool is a general-purpose HTTP/REST client for agents.
// It allows agents to make HTTP requests to external APIs with SSRF protection.
//
// Input examples:
//
//	{"method": "GET", "url": "https://api.example.com/data"}
//	{"method": "POST", "url": "https://api.example.com/data", "body": {"key": "value"}, "headers": {"Authorization": "Bearer xxx"}}
type HTTPTool struct {
	BaseTool
	BaseURL          string            // Optional base URL prefix
	Headers          map[string]string // Default headers applied to all requests
	Timeout          time.Duration
	ResponseMaxBytes int               // Maximum response body size (0 = use default 5MB)
	httpClient       *http.Client
}

// NewHTTPTool creates a general-purpose HTTP client tool.
func NewHTTPTool(opts ...func(*HTTPTool)) *HTTPTool {
	t := &HTTPTool{
		BaseTool: BaseTool{
			NameValue:        "HTTPTool",
			DescriptionValue: "Make HTTP requests to REST APIs. Input: {'method': 'GET/POST/PUT/DELETE/PATCH', 'url': '...', 'body': {...}, 'headers': {...}, 'query': {...}}. Returns response body.",
		},
		Headers:          make(map[string]string),
		Timeout:          30 * time.Second,
		ResponseMaxBytes: 5 * 1024 * 1024,
	}
	for _, opt := range opts {
		opt(t)
	}
	t.httpClient = &http.Client{Timeout: t.Timeout}
	return t
}

// WithHTTPBaseURL sets a base URL prefix for all requests.
func WithHTTPBaseURL(baseURL string) func(*HTTPTool) {
	return func(t *HTTPTool) {
		t.BaseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHTTPHeaders sets default headers for all requests.
func WithHTTPHeaders(headers map[string]string) func(*HTTPTool) {
	return func(t *HTTPTool) {
		for k, v := range headers {
			t.Headers[k] = v
		}
	}
}

// WithHTTPTimeout sets the request timeout.
func WithHTTPTimeout(timeout time.Duration) func(*HTTPTool) {
	return func(t *HTTPTool) {
		t.Timeout = timeout
	}
}

// WithHTTPMaxBytes sets the maximum response body size.
func WithHTTPMaxBytes(n int) func(*HTTPTool) {
	return func(t *HTTPTool) {
		t.ResponseMaxBytes = n
	}
}

func (t *HTTPTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	method, _ := input["method"].(string)
	url, _ := input["url"].(string)
	if method == "" {
		method = "GET"
	}
	if url == "" {
		return "", fmt.Errorf("'url' is required")
	}

	// Prepend base URL if set
	if t.BaseURL != "" && !strings.HasPrefix(url, "http") {
		url = t.BaseURL + "/" + strings.TrimLeft(url, "/")
	}

	// Validate the URL against SSRF and egress restrictions.
	if err := t.validateURL(url); err != nil {
		return "", fmt.Errorf("http request blocked: %w", err)
	}

	// Build query parameters
	if queryParams, ok := input["query"].(map[string]interface{}); ok {
		params := make([]string, 0, len(queryParams))
		for k, v := range queryParams {
			params = append(params, fmt.Sprintf("%s=%v", k, v))
		}
		if len(params) > 0 {
			sep := "?"
			if strings.Contains(url, "?") {
				sep = "&"
			}
			url += sep + strings.Join(params, "&")
		}
	}

	// Build request body
	var reqBody io.Reader
	if body, ok := input["body"]; ok {
		data, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("failed to marshal body: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), url, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Apply default headers
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}

	// Apply per-request headers
	if headers, ok := input["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	if req.Header.Get("Content-Type") == "" && reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Resolve redirects manually to validate each hop against SSRF rules.
	t.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		if err := t.validateURL(req.URL.String()); err != nil {
			return fmt.Errorf("redirect blocked: %w", err)
		}
		return http.ErrUseLastResponse
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	maxBytes := t.ResponseMaxBytes
	if maxBytes <= 0 {
		maxBytes = 5 * 1024 * 1024
	}
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)))

	// Build structured response
	result := map[string]interface{}{
		"status":      resp.StatusCode,
		"status_text": resp.Status,
	}

	// Try to parse as JSON for pretty output
	var jsonResp interface{}
	if json.Unmarshal(respBody, &jsonResp) == nil {
		result["body"] = jsonResp
	} else {
		result["body"] = string(respBody)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return string(output), nil
}

func (t *HTTPTool) RequiresReview() bool { return true }
func (t *HTTPTool) Name() string { return t.BaseTool.NameValue }
func (t *HTTPTool) Description() string { return t.BaseTool.DescriptionValue }

// validateURL checks the target URL against SSRF and egress restrictions.
func (t *HTTPTool) validateURL(rawURL string) error {
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

	if t.ResponseMaxBytes > 0 && t.ResponseMaxBytes < 1024 {
		return fmt.Errorf("response_max_bytes must be at least 1024, got %d", t.ResponseMaxBytes)
	}

	return nil
}

func (t *HTTPTool) isBlockedIP(ip net.IP) bool {
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
