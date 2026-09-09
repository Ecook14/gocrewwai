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

// JiraTool allows agents to interact with Jira.
type JiraTool struct {
	BaseTool
	client  *http.Client
	baseURL string
	email   string
	token   string
}

// NewJiraTool creates a Jira integration tool.
func NewJiraTool(baseURL, email, token string) *JiraTool {
	if baseURL == "" {
		baseURL = os.Getenv("JIRA_BASE_URL")
	}
	if email == "" {
		email = os.Getenv("JIRA_EMAIL")
	}
	if token == "" {
		token = os.Getenv("JIRA_TOKEN")
	}
	return &JiraTool{
		BaseTool: BaseTool{
			NameValue:        "JiraTool",
			DescriptionValue: "Interacts with Jira. Actions: search_issues, create_issue, update_issue, get_issue, list_projects.",
		},
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: baseURL,
		email:   email,
		token:   token,
	}
}

func (j *JiraTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, ok := input["action"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'action' parameter")
	}

	switch action {
	case "search_issues":
		query, _ := input["query"].(string)
		return j.searchIssues(ctx, query)
	case "create_issue":
		project, _ := input["project"].(string)
		summary, _ := input["summary"].(string)
		issueType, _ := input["issue_type"].(string)
		return j.createIssue(ctx, project, summary, issueType)
	case "update_issue":
		key, _ := input["key"].(string)
		updates, _ := input["updates"].(map[string]interface{})
		return j.updateIssue(ctx, key, updates)
	case "get_issue":
		key, _ := input["key"].(string)
		return j.getIssue(ctx, key)
	case "list_projects":
		return j.listProjects(ctx)
	default:
		return "", fmt.Errorf("unknown Jira action: %s", action)
	}
}

func (j *JiraTool) searchIssues(ctx context.Context, query string) (string, error) {
	url := fmt.Sprintf("%s/rest/api/3/search?jql=%s&maxResults=10", j.baseURL, strings.ReplaceAll(query, " ", "+"))
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(j.email, j.token)
	resp, err := j.client.Do(req.WithContext(ctx))
	if err != nil { return "", err }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (j *JiraTool) createIssue(ctx context.Context, project, summary, issueType string) (string, error) {
	if issueType == "" { issueType = "Task" }
	data, _ := json.Marshal(map[string]interface{}{
		"fields": map[string]interface{}{
			"project": map[string]string{"key": project},
			"summary": summary,
			"issuetype": map[string]string{"name": issueType},
		},
	})
	url := fmt.Sprintf("%s/rest/api/3/issue", j.baseURL)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(data)))
	req.SetBasicAuth(j.email, j.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := j.client.Do(req.WithContext(ctx))
	if err != nil { return "", err }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (j *JiraTool) updateIssue(ctx context.Context, key string, updates map[string]interface{}) (string, error) {
	data, _ := json.Marshal(map[string]interface{}{"fields": updates})
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", j.baseURL, key)
	req, _ := http.NewRequest("PUT", url, strings.NewReader(string(data)))
	req.SetBasicAuth(j.email, j.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := j.client.Do(req.WithContext(ctx))
	if err != nil { return "", err }
	defer resp.Body.Close()
	return fmt.Sprintf("Updated issue %s", key), nil
}

func (j *JiraTool) getIssue(ctx context.Context, key string) (string, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", j.baseURL, key)
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(j.email, j.token)
	resp, err := j.client.Do(req.WithContext(ctx))
	if err != nil { return "", err }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (j *JiraTool) listProjects(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/rest/api/3/project/search?maxResults=10", j.baseURL)
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(j.email, j.token)
	resp, err := j.client.Do(req.WithContext(ctx))
	if err != nil { return "", err }
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (j *JiraTool) RequiresReview() bool { return true }
func (j *JiraTool) Name() string { return j.BaseTool.NameValue }
func (j *JiraTool) Description() string { return j.BaseTool.DescriptionValue }
