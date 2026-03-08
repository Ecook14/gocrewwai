package tools

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/v60/github"
)

// GitHubTool allows agents to interact with GitHub.
type GitHubTool struct {
	client *github.Client
}

func NewGitHubTool(token string) *GitHubTool {
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return nil
	}
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	client := github.NewClient(httpClient).WithAuthToken(token)
	return &GitHubTool{client: client}
}

func (t *GitHubTool) Name() string { return "GitHubTool" }

func (t *GitHubTool) Description() string {
	return "Interacts with GitHub. Actions: search_repos, create_issue, get_repo_info."
}

func (t *GitHubTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, ok := input["action"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'action'")
	}

	switch action {
	case "search_repos":
		query, _ := input["query"].(string)
		result, _, err := t.client.Search.Repositories(ctx, query, nil)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Found %d repositories.", *result.Total), nil
	case "create_issue":
		owner, _ := input["owner"].(string)
		repo, _ := input["repo"].(string)
		title, _ := input["title"].(string)
		body, _ := input["body"].(string)
		issue, _, err := t.client.Issues.Create(ctx, owner, repo, &github.IssueRequest{
			Title: &title,
			Body:  &body,
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Issue created: %s", *issue.HTMLURL), nil
	default:
		return "", fmt.Errorf("unsupported github action: %s", action)
	}
}

func (t *GitHubTool) RequiresReview() bool { return true }
