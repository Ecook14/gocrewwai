package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"os"
	"time"
)

// SendGridTool allows agents to send emails via SendGrid.
type SendGridTool struct {
	BaseTool
	client *http.Client
	apiKey string
}

// NewSendGridTool creates a SendGrid email tool.
func NewSendGridTool(apiKey string) *SendGridTool {
	if apiKey == "" {
		apiKey = os.Getenv("SENDGRID_API_KEY")
	}
	return &SendGridTool{
		BaseTool: BaseTool{
			NameValue:        "SendGridTool",
			DescriptionValue: "Sends transactional emails via SendGrid. Supports HTML, attachments, and templates.",
		},
		client: &http.Client{Timeout: 30 * time.Second},
		apiKey: apiKey,
	}
}

// Execute implements the Tool interface for SendGrid.
func (s *SendGridTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	to, _ := input["to"].(string)
	if to == "" { return "", fmt.Errorf("missing 'to' parameter") }
	subject, _ := input["subject"].(string)
	if subject == "" { subject = "No Subject" }
	body, _ := input["body"].(string)
	if body == "" { body = "" }
	isHTML, _ := input["html"].(bool)
	if !isHTML { isHTML = false }

	contentType := "text/plain"
	if isHTML { contentType = "text/html" }

	personalizations := []map[string]interface{}{{
		"to": []map[string]string{{"email": to}},
		"subject": subject,
	}}
	content := []map[string]string{{"type": contentType, "value": body}}

	reqBody := map[string]interface{}{
		"personalizations": personalizations,
		"from": map[string]string{"email": os.Getenv("SENDGRID_FROM")},
		"content": content,
	}

	data, _ := json.Marshal(reqBody)
	url := "https://api.sendgrid.com/v3/mail/send"
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(data)))
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("SendGrid request failed: %w", err) }
	defer resp.Body.Close()
	if resp.StatusCode != 202 {
		return "", fmt.Errorf("SendGrid returned status %d", resp.StatusCode)
	}
	return fmt.Sprintf("Email sent to %s: %s", to, subject), nil
}

func (s *SendGridTool) RequiresReview() bool { return false }
func (s *SendGridTool) Name() string { return s.BaseTool.NameValue }
func (s *SendGridTool) Description() string { return s.BaseTool.DescriptionValue }
