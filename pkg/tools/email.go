package tools

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// EmailTool allows agents to send emails via SMTP.
//
// Input example:
//
//	{"to": "user@example.com", "subject": "Report", "body": "Here is the report.", "cc": "manager@example.com"}
type EmailTool struct {
	BaseTool
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
}

// NewEmailTool creates an SMTP email sending tool.
func NewEmailTool(smtpHost string, smtpPort int, username, password, from string) *EmailTool {
	return &EmailTool{
		BaseTool: BaseTool{
			NameValue:        "EmailTool",
			DescriptionValue: "Send emails via SMTP. Input: {'to': 'recipient@example.com', 'subject': 'Subject', 'body': 'Email body', 'cc': 'optional@cc.com'}. Supports plain text emails.",
		},
		SMTPHost: smtpHost,
		SMTPPort: smtpPort,
		Username: username,
		Password: password,
		From:     from,
	}
}

func (t *EmailTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	to, _ := input["to"].(string)
	subject, _ := input["subject"].(string)
	body, _ := input["body"].(string)

	if to == "" {
		return "", fmt.Errorf("'to' is required")
	}
	if subject == "" {
		return "", fmt.Errorf("'subject' is required")
	}

	// Build recipients list
	recipients := []string{to}
	cc, _ := input["cc"].(string)
	if cc != "" {
		recipients = append(recipients, strings.Split(cc, ",")...)
	}

	// Build email message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", t.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	if cc != "" {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", cc))
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	addr := fmt.Sprintf("%s:%d", t.SMTPHost, t.SMTPPort)
	auth := smtp.PlainAuth("", t.Username, t.Password, t.SMTPHost)

	err := smtp.SendMail(addr, auth, t.From, recipients, []byte(msg.String()))
	if err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	return fmt.Sprintf("Email sent successfully to %s (subject: %s)", to, subject), nil
}

func (t *EmailTool) RequiresReview() bool { return true } // Emails should always be reviewed
