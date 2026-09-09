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

// LinearTool allows agents to interact with Linear.
type LinearTool struct {
	BaseTool
	client  *http.Client
	baseURL string
	token   string
}

// NewLinearTool creates a Linear project management tool.
func NewLinearTool(token string) *LinearTool {
	if token == "" {
		token = os.Getenv("LINEAR_TOKEN")
	}
	return &LinearTool{
		BaseTool: BaseTool{
			NameValue:        "LinearTool",
			DescriptionValue: "Interacts with Linear. Actions: search_issues, create_issue, get_issue, list_projects, update_issue.",
		},
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://api.linear.app/graphql",
		token:   token,
	}
}

// Execute implements the Tool interface for Linear.
func (l *LinearTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, ok := input["action"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'action' parameter")
	}

	switch action {
	case "search_issues":
		query, _ := input["query"].(string)
		return l.searchIssues(ctx, query)
	case "create_issue":
		teamID, _ := input["team_id"].(string)
		title, _ := input["title"].(string)
		description, _ := input["description"].(string)
		return l.createIssue(ctx, teamID, title, description)
	case "get_issue":
		issueID, _ := input["issue_id"].(string)
		return l.getIssue(ctx, issueID)
	case "list_projects":
		return l.listProjects(ctx)
	case "update_issue":
		issueID, _ := input["issue_id"].(string)
		updates, _ := input["updates"].(map[string]interface{})
		return l.updateIssue(ctx, issueID, updates)
	default:
		return "", fmt.Errorf("unknown Linear action: %s", action)
	}
}

func (l *LinearTool) query(query string, variables map[string]interface{}) (string, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{"query": query, "variables": variables})
	req, _ := http.NewRequest("POST", l.baseURL, strings.NewReader(string(reqBody)))
	req.Header.Set("Authorization", "Bearer "+l.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := l.client.Do(req.WithContext(context.Background()))
	if err != nil { return "", fmt.Errorf("Linear request failed: %w", err) }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (l *LinearTool) searchIssues(ctx context.Context, query string) (string, error) {
	ql := `query($query: String!) { issues(filter: {title: {like: $query}}, limit: 10) { nodes { id title description status { name } } } }`
	return l.query(ql, map[string]interface{}{"query": query})
}

func (l *LinearTool) createIssue(ctx context.Context, teamID, title, description string) (string, error) {
	ql := `mutation($teamId: String!, $title: String!, $description: String!) { issueCreate(teamId: $teamId, title: $title, description: $description) { id title } }`
	return l.query(ql, map[string]interface{}{"teamId": teamID, "title": title, "description": description})
}

func (l *LinearTool) getIssue(ctx context.Context, issueID string) (string, error) {
	ql := `query($id: String!) { issue(id: $id) { id title description status { name } assignees { name } } }`
	return l.query(ql, map[string]interface{}{"id": issueID})
}

func (l *LinearTool) listProjects(ctx context.Context) (string, error) {
	ql := `query { projects(first: 10) { nodes { id name description } } }`
	return l.query(ql, map[string]interface{}{})
}

func (l *LinearTool) updateIssue(ctx context.Context, issueID string, updates map[string]interface{}) (string, error) {
	ql := `mutation($id: String!, $updates: IssueUpdateInput!) { issueUpdate(id: $id, updates: $updates) { id title } }`
	return l.query(ql, map[string]interface{}{"id": issueID, "updates": updates})
}

func (l *LinearTool) RequiresReview() bool { return true }
func (l *LinearTool) Name() string { return l.BaseTool.NameValue }
func (l *LinearTool) Description() string { return l.BaseTool.DescriptionValue }
