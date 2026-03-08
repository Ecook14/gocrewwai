package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPTool is a general-purpose HTTP/REST client for agents.
// It allows agents to make arbitrary HTTP requests to external APIs.
//
// Input examples:
//
//	{"method": "GET", "url": "https://api.example.com/data"}
//	{"method": "POST", "url": "https://api.example.com/data", "body": {"key": "value"}, "headers": {"Authorization": "Bearer xxx"}}
type HTTPTool struct {
	BaseTool
	BaseURL    string            // Optional base URL prefix
	Headers    map[string]string // Default headers applied to all requests
	Timeout    time.Duration
	httpClient *http.Client
}

// NewHTTPTool creates a general-purpose HTTP client tool.
func NewHTTPTool(opts ...func(*HTTPTool)) *HTTPTool {
	t := &HTTPTool{
		BaseTool: BaseTool{
			NameValue:        "HTTPTool",
			DescriptionValue: "Make HTTP requests to REST APIs. Input: {'method': 'GET/POST/PUT/DELETE/PATCH', 'url': '...', 'body': {...}, 'headers': {...}, 'query': {...}}. Returns response body.",
		},
		Headers: make(map[string]string),
		Timeout: 30 * time.Second,
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

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // 5MB limit

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
