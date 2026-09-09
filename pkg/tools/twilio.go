package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// TwilioTool allows agents to send SMS and make phone calls.
type TwilioTool struct {
	BaseTool
	client     *http.Client
	accountSID string
	authToken  string
}

// NewTwilioTool creates a Twilio communication tool.
func NewTwilioTool(accountSID, authToken string) *TwilioTool {
	if accountSID == "" {
		accountSID = os.Getenv("TWILIO_ACCOUNT_SID")
	}
	if authToken == "" {
		authToken = os.Getenv("TWILIO_AUTH_TOKEN")
	}
	return &TwilioTool{
		BaseTool: BaseTool{
			NameValue:        "TwilioTool",
			DescriptionValue: "Sends SMS, makes calls, and sends WhatsApp messages via Twilio.",
		},
		client:     &http.Client{Timeout: 30 * time.Second},
		accountSID: accountSID,
		authToken:  authToken,
	}
}

// Execute implements the Tool interface for Twilio.
func (t *TwilioTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, ok := input["action"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'action' parameter")
	}

	switch action {
	case "send_sms":
		to, _ := input["to"].(string)
		body, _ := input["body"].(string)
		return t.sendSMS(ctx, to, body)
	case "make_call":
		to, _ := input["to"].(string)
		url_, _ := input["twiml_url"].(string)
		return t.makeCall(ctx, to, url_)
	case "send_whatsapp":
		to, _ := input["to"].(string)
		body, _ := input["body"].(string)
		return t.sendWhatsApp(ctx, to, body)
	default:
		return "", fmt.Errorf("unknown Twilio action: %s", action)
	}
}

func (t *TwilioTool) sendSMS(ctx context.Context, to, body string) (string, error) {
	data := map[string]string{"To": to, "Body": body, "From": os.Getenv("TWILIO_PHONE_NUMBER")}
	reqBody, _ := json.Marshal(data)
	url := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(reqBody)))
	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Twilio SMS failed: %w", err) }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (t *TwilioTool) makeCall(ctx context.Context, to, twimlURL string) (string, error) {
	data := map[string]string{"To": to, "From": os.Getenv("TWILIO_PHONE_NUMBER"), "Url": twimlURL}
	reqBody, _ := json.Marshal(data)
	url := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Calls.json", t.accountSID)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(reqBody)))
	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Twilio call failed: %w", err) }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (t *TwilioTool) sendWhatsApp(ctx context.Context, to, body string) (string, error) {
	data := map[string]string{"To": "whatsapp:" + to, "Body": body, "From": os.Getenv("TWILIO_WHATSAPP_NUMBER")}
	reqBody, _ := json.Marshal(data)
	url := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(reqBody)))
	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Twilio WhatsApp failed: %w", err) }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (t *TwilioTool) RequiresReview() bool { return false }
func (t *TwilioTool) Name() string { return t.BaseTool.NameValue }
func (t *TwilioTool) Description() string { return t.BaseTool.DescriptionValue }
