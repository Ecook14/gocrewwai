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

// DiscordTool allows agents to send messages and interact with Discord.
type DiscordTool struct {
	BaseTool
	client    *http.Client
	botToken  string
	channelID string
}

// NewDiscordTool creates a Discord integration tool.
func NewDiscordTool(botToken, channelID string) *DiscordTool {
	if botToken == "" {
		botToken = os.Getenv("DISCORD_BOT_TOKEN")
	}
	return &DiscordTool{
		BaseTool: BaseTool{
			NameValue:        "DiscordTool",
			DescriptionValue: "Sends messages and interacts with Discord channels. Actions: send_message, delete_message, edit_message.",
		},
		client: &http.Client{Timeout: 15 * time.Second},
		botToken: botToken,
		channelID: channelID,
	}
}

// discordMessage represents a Discord message payload.
type discordMessage struct {
	Content string `json:"content"`
	TTS     bool   `json:"tts,omitempty"`
}

// Execute implements the Tool interface for Discord.
func (d *DiscordTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, ok := input["action"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'action' parameter")
	}

	switch action {
	case "send_message":
		msg, _ := input["message"].(string)
		if msg == "" { return "", fmt.Errorf("missing 'message' parameter") }
		return d.sendMessage(ctx, msg)
	case "delete_message":
		msgID, _ := input["message_id"].(string)
		return d.deleteMessage(ctx, msgID)
	case "edit_message":
		msgID, _ := input["message_id"].(string)
		newMsg, _ := input["new_message"].(string)
		return d.editMessage(ctx, msgID, newMsg)
	default:
		return "", fmt.Errorf("unknown Discord action: %s", action)
	}
}

func (d *DiscordTool) sendMessage(ctx context.Context, message string) (string, error) {
	url := fmt.Sprintf("https://discord.com/api/v9/channels/%s/messages", d.channelID)
	data, _ := json.Marshal(discordMessage{Content: message})
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(data)))
	req.Header.Set("Authorization", "Bot "+d.botToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Discord send failed: %w", err) }
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return "", fmt.Errorf("Discord returned status %d", resp.StatusCode)
	}
	return fmt.Sprintf("Message sent to channel %s", d.channelID), nil
}

func (d *DiscordTool) deleteMessage(ctx context.Context, msgID string) (string, error) {
	url := fmt.Sprintf("https://discord.com/api/v9/channels/%s/messages/%s", d.channelID, msgID)
	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Authorization", "Bot "+d.botToken)
	resp, err := d.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Discord delete failed: %w", err) }
	defer resp.Body.Close()
	return fmt.Sprintf("Message %s deleted", msgID), nil
}

func (d *DiscordTool) editMessage(ctx context.Context, msgID, newMessage string) (string, error) {
	url := fmt.Sprintf("https://discord.com/api/v9/channels/%s/messages/%s", d.channelID, msgID)
	data, _ := json.Marshal(discordMessage{Content: newMessage})
	req, _ := http.NewRequest("PATCH", url, strings.NewReader(string(data)))
	req.Header.Set("Authorization", "Bot "+d.botToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.client.Do(req.WithContext(ctx))
	if err != nil { return "", fmt.Errorf("Discord edit failed: %w", err) }
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Discord returned status %d", resp.StatusCode)
	}
	return fmt.Sprintf("Message %s edited", msgID), nil
}

func (d *DiscordTool) RequiresReview() bool { return false }
func (d *DiscordTool) Name() string { return d.BaseTool.NameValue }
func (d *DiscordTool) Description() string { return d.BaseTool.DescriptionValue }
