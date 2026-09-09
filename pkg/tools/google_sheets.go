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

// GoogleSheetsTool allows agents to interact with Google Sheets.
type GoogleSheetsTool struct {
	BaseTool
	client *http.Client
	token  string
}

// NewGoogleSheetsTool creates a Google Sheets integration tool.
func NewGoogleSheetsTool(token string) *GoogleSheetsTool {
	if token == "" {
		token = os.Getenv("GOOGLE_SHEETS_TOKEN")
	}
	return &GoogleSheetsTool{
		BaseTool: BaseTool{
			NameValue:        "GoogleSheetsTool",
			DescriptionValue: "Interacts with Google Sheets. Actions: read_range, append_rows, update_cells, clear_range.",
		},
		client: &http.Client{Timeout: 30 * time.Second},
		token:  token,
	}
}

// Execute implements the Tool interface for Google Sheets.
func (g *GoogleSheetsTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, ok := input["action"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'action' parameter")
	}

	spreadsheetID, _ := input["spreadsheet_id"].(string)
	if spreadsheetID == "" {
		return "", fmt.Errorf("missing 'spreadsheet_id' parameter")
	}

	switch action {
	case "read_range":
		rangeName, _ := input["range"].(string)
		return g.readRange(ctx, spreadsheetID, rangeName)
	case "append_rows":
		rangeName, _ := input["range"].(string)
		values, _ := input["values"].([][]interface{})
		return g.appendRows(ctx, spreadsheetID, rangeName, values)
	case "update_cells":
		rangeName, _ := input["range"].(string)
		values, _ := input["values"].([][]interface{})
		return g.updateCells(ctx, spreadsheetID, rangeName, values)
	case "clear_range":
		rangeName, _ := input["range"].(string)
		return g.clearRange(ctx, spreadsheetID, rangeName)
	default:
		return "", fmt.Errorf("unknown Google Sheets action: %s", action)
	}
}

func (g *GoogleSheetsTool) url(spreadsheetID string) string {
	return fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s", spreadsheetID)
}

func (g *GoogleSheetsTool) readRange(ctx context.Context, spreadsheetID, rangeName string) (string, error) {
	url := g.url(spreadsheetID) + "/values/" + rangeName
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+g.token)
	resp, err := g.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Google Sheets read failed: %w", err) }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (g *GoogleSheetsTool) appendRows(ctx context.Context, spreadsheetID, rangeName string, values [][]interface{}) (string, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{"values": values})
	url := g.url(spreadsheetID) + "/values/" + rangeName + ":append?valueInputOption=RAW"
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(reqBody)))
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Google Sheets append failed: %w", err) }
	defer resp.Body.Close()
	return fmt.Sprintf("Appended %d rows to %s", len(values), rangeName), nil
}

func (g *GoogleSheetsTool) updateCells(ctx context.Context, spreadsheetID, rangeName string, values [][]interface{}) (string, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{"values": values})
	url := g.url(spreadsheetID) + "/values/" + rangeName + "?valueInputOption=RAW"
	req, _ := http.NewRequest("PUT", url, strings.NewReader(string(reqBody)))
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Google Sheets update failed: %w", err) }
	defer resp.Body.Close()
	return fmt.Sprintf("Updated %d cells in %s", len(values), rangeName), nil
}

func (g *GoogleSheetsTool) clearRange(ctx context.Context, spreadsheetID, rangeName string) (string, error) {
	url := g.url(spreadsheetID) + "/values/" + rangeName + ":clear"
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+g.token)
	resp, err := g.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Google Sheets clear failed: %w", err) }
	defer resp.Body.Close()
	return fmt.Sprintf("Cleared range %s", rangeName), nil
}

func (g *GoogleSheetsTool) RequiresReview() bool { return true }
func (g *GoogleSheetsTool) Name() string { return g.BaseTool.NameValue }
func (g *GoogleSheetsTool) Description() string { return g.BaseTool.DescriptionValue }
