package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/slack-go/slack"
)

// SlackTool allows agents to send messages to Slack.
type SlackTool struct {
	BaseTool
	client *slack.Client
}

func NewSlackTool(token string) *SlackTool {
	if token == "" {
		token = os.Getenv("SLACK_TOKEN")
	}
	if token == "" {
		return nil
	}
	return &SlackTool{
		BaseTool: BaseTool{
			NameValue:        "SlackTool",
			DescriptionValue: "Sends messages to Slack channels. Action: post_message.",
		},
		client: slack.New(token),
	}
}


func (t *SlackTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, ok := input["action"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'action'")
	}

	switch action {
	case "post_message":
		channel, _ := input["channel"].(string)
		text, _ := input["text"].(string)
		_, _, err := t.client.PostMessage(channel, slack.MsgOptionText(text, false))
		if err != nil {
			return "", err
		}
		return "Message posted to Slack.", nil
	default:
		return "", fmt.Errorf("unsupported slack action: %s", action)
	}
}

func (t *SlackTool) RequiresReview() bool { return true }
