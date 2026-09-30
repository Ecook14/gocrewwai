package tools

import (
	"context"
	"strings"
	"testing"
)

func TestJSONToolNegativeIndexRejected(t *testing.T) {
	jt := NewJSONTool()
	for _, path := range []string{"users.-1", "users.1abc", "users.99"} {
		_, err := jt.Execute(context.Background(), map[string]interface{}{
			"action": "query",
			"data":   `{"users":[{"name":"a"}]}`,
			"path":   path,
		})
		if err == nil || !strings.Contains(err.Error(), "invalid array index") {
			t.Errorf("path %q: got %v, want invalid array index", path, err)
		}
	}
	if _, err := jt.Execute(context.Background(), map[string]interface{}{
		"action": "query",
		"data":   `{"users":[{"name":"a"}]}`,
		"path":   "users.0.name",
	}); err != nil {
		t.Errorf("valid index rejected: %v", err)
	}
}

func TestJSONToolDepthAndSizeBudgets(t *testing.T) {
	jt := NewJSONTool()
	deep := strings.Repeat(`{"a":`, 200) + `1` + strings.Repeat(`}`, 200)
	if _, err := jt.Execute(context.Background(), map[string]interface{}{"action": "parse", "data": deep}); err == nil {
		t.Error("deeply nested JSON must be rejected")
	}
	big := `"` + strings.Repeat("x", maxJSONInputBytes+1) + `"`
	if _, err := jt.Execute(context.Background(), map[string]interface{}{"action": "parse", "data": big}); err == nil {
		t.Error("oversize JSON input must be rejected")
	}
}

func TestRegexReplaceBudget(t *testing.T) {
	rt := NewRegexTool()
	_, err := rt.Execute(context.Background(), map[string]interface{}{
		"action": "replace", "pattern": "a", "text": "aaa",
		"replacement": strings.Repeat("b", maxRegexReplBytes+1),
	})
	if err == nil {
		t.Error("oversize replacement must be rejected")
	}
}

func TestEmailHeaderInjectionRejected(t *testing.T) {
	et := NewEmailTool("127.0.0.1", 1, "u", "p", "from@example.com")
	for _, subject := range []string{"hi\r\nReply-To: evil@x.com", "hi\nBcc: evil@x.com"} {
		_, err := et.Execute(context.Background(), map[string]interface{}{
			"to": "to@example.com", "subject": subject, "body": "b",
		})
		if err == nil || !strings.Contains(err.Error(), "single-line") {
			t.Errorf("subject %q: got %v, want single-line rejection", subject, err)
		}
	}
	_, err := et.Execute(context.Background(), map[string]interface{}{
		"to": "not-an-address", "subject": "ok", "body": "b",
	})
	if err == nil {
		t.Error("invalid to address must be rejected")
	}
}

func TestResourceIDValidators(t *testing.T) {
	if !validDiscordID("123456789") || validDiscordID("12%2f34") || validDiscordID("abc") || validDiscordID("") {
		t.Error("validDiscordID wrong")
	}
	if !validJiraKey("PROJ-123") || validJiraKey("PROJ-1/x") || validJiraKey("") {
		t.Error("validJiraKey wrong")
	}
	if !validHubSpotID("12345") || validHubSpotID("a/b") || validHubSpotID("") {
		t.Error("validHubSpotID wrong")
	}
	if !validSheetsID("1BxiMVs0XRA5n") || validSheetsID("a/b") || validSheetsID("a%2fb") {
		t.Error("validSheetsID wrong")
	}
	if !validSheetsRange("Sheet1!A1:B2") || validSheetsRange("A#/B") || validSheetsRange("") {
		t.Error("validSheetsRange wrong")
	}
	if !validESIndex("logs-2024.01") || validESIndex("-bad") || validESIndex("a#b") || validESIndex("a/b") || validESIndex("") {
		t.Error("validESIndex wrong")
	}
	if !validESID("doc-1") || validESID("a/b") || validESID("") {
		t.Error("validESID wrong")
	}
}
