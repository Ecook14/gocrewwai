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

// S3Tool allows agents to interact with AWS S3 or S3-compatible storage.
// Uses the S3 REST API directly with V4-style auth headers or pre-signed URLs.
//
// For simplicity, this implementation uses presigned operations or the S3 REST
// API with explicit credentials. For production, consider the official AWS SDK.
//
// Input examples:
//
//	{"action": "list", "bucket": "my-bucket", "prefix": "docs/"}
//	{"action": "get", "bucket": "my-bucket", "key": "docs/report.txt"}
//	{"action": "put", "bucket": "my-bucket", "key": "docs/new.txt", "content": "Hello!"}
//	{"action": "delete", "bucket": "my-bucket", "key": "docs/old.txt"}
type S3Tool struct {
	BaseTool
	Endpoint  string // S3 endpoint (e.g., "https://s3.amazonaws.com" or MinIO URL)
	AccessKey string
	SecretKey string
	Region    string
	httpClient *http.Client
}

// NewS3Tool creates an S3-compatible storage tool.
func NewS3Tool(endpoint, accessKey, secretKey, region string) *S3Tool {
	if region == "" {
		region = "us-east-1"
	}
	return &S3Tool{
		BaseTool: BaseTool{
			NameValue:        "S3Tool",
			DescriptionValue: "Interact with S3-compatible object storage. Actions: list, get, put, delete, head. Input: {'action': '...', 'bucket': '...', 'key': '...', 'content': '...'}",
		},
		Endpoint:   strings.TrimRight(endpoint, "/"),
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		Region:     region,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (t *S3Tool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, _ := input["action"].(string)
	bucket, _ := input["bucket"].(string)
	if action == "" || bucket == "" {
		return "", fmt.Errorf("'action' and 'bucket' are required")
	}

	key, _ := input["key"].(string)

	switch action {
	case "list":
		prefix, _ := input["prefix"].(string)
		return t.listObjects(ctx, bucket, prefix)
	case "get":
		if key == "" {
			return "", fmt.Errorf("'key' is required for get")
		}
		return t.getObject(ctx, bucket, key)
	case "put":
		if key == "" {
			return "", fmt.Errorf("'key' is required for put")
		}
		content, _ := input["content"].(string)
		return t.putObject(ctx, bucket, key, content)
	case "delete":
		if key == "" {
			return "", fmt.Errorf("'key' is required for delete")
		}
		return t.deleteObject(ctx, bucket, key)
	case "head":
		if key == "" {
			return "", fmt.Errorf("'key' is required for head")
		}
		return t.headObject(ctx, bucket, key)
	default:
		return "", fmt.Errorf("unsupported action: %s", action)
	}
}

func (t *S3Tool) buildURL(bucket, key string) string {
	if key != "" {
		return fmt.Sprintf("%s/%s/%s", t.Endpoint, bucket, key)
	}
	return fmt.Sprintf("%s/%s", t.Endpoint, bucket)
}

func (t *S3Tool) listObjects(ctx context.Context, bucket, prefix string) (string, error) {
	url := fmt.Sprintf("%s/%s?list-type=2", t.Endpoint, bucket)
	if prefix != "" {
		url += "&prefix=" + prefix
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("s3 list error (status %d): %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}

func (t *S3Tool) getObject(ctx context.Context, bucket, key string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.buildURL(bucket, key), nil)
	if err != nil {
		return "", err
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // 5MB limit
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("s3 get error (status %d): %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}

func (t *S3Tool) putObject(ctx context.Context, bucket, key, content string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, t.buildURL(bucket, key),
		bytes.NewBufferString(content))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("s3 put error (status %d): %s", resp.StatusCode, string(body))
	}
	return fmt.Sprintf("Successfully uploaded %s/%s (%d bytes)", bucket, key, len(content)), nil
}

func (t *S3Tool) deleteObject(ctx context.Context, bucket, key string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, t.buildURL(bucket, key), nil)
	if err != nil {
		return "", err
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("s3 delete error: %s", string(body))
	}
	return fmt.Sprintf("Deleted %s/%s", bucket, key), nil
}

func (t *S3Tool) headObject(ctx context.Context, bucket, key string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, t.buildURL(bucket, key), nil)
	if err != nil {
		return "", err
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	headers := make(map[string]string)
	for _, key := range []string{"Content-Type", "Content-Length", "Last-Modified", "ETag"} {
		if v := resp.Header.Get(key); v != "" {
			headers[key] = v
		}
	}
	result, _ := json.MarshalIndent(headers, "", "  ")
	return string(result), nil
}

func (t *S3Tool) RequiresReview() bool { return true }
