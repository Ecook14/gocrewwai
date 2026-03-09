package tools

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

// BrowserTool Action types
const (
	ActionNavigate   = "navigate"
	ActionClick      = "click"
	ActionType       = "type"
	ActionScreenshot = "screenshot"
	ActionGetText    = "get_text"
)

// BrowserTool provides direct headless browser control using chromedp.
type BrowserTool struct {
	BaseTool
	// Timeout for browser actions. Defaults to 30 seconds.
	Timeout time.Duration
}

func NewBrowserTool() *BrowserTool {
	return &BrowserTool{
		BaseTool: BaseTool{
			NameValue: "BrowserControl",
			DescriptionValue: `Controls a headless browser. Supported actions:
- navigate: {"url": "https://example.com"}
- click: {"selector": "#button-id"}
- type: {"selector": "input", "text": "hello"}
- screenshot: {"filename": "output.png"}
- get_text: {"selector": "body"}`,
		},
		Timeout: 30 * time.Second,
	}
}


func (t *BrowserTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	action, ok := input["action"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'action' parameter")
	}

	// Create browser context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.DisableGPU,
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	// Add timeout
	ctx, cancelTimeout := context.WithTimeout(taskCtx, t.Timeout)
	defer cancelTimeout()

	var result string

	switch action {
	case ActionNavigate:
		url, _ := input["url"].(string)
		if url == "" {
			return "", fmt.Errorf("navigate requires 'url'")
		}
		// Enhanced: Wait for a specific selector if provided, otherwise just navigate
		waitSelector, _ := input["wait_selector"].(string)
		var actions []chromedp.Action
		actions = append(actions, chromedp.Navigate(url))
		if waitSelector != "" {
			actions = append(actions, chromedp.WaitVisible(waitSelector, chromedp.ByQuery))
		}
		
		err := chromedp.Run(ctx, actions...)
		if err != nil {
			return "", err
		}
		result = fmt.Sprintf("Navigated to %s", url)

	case ActionClick:
		selector, _ := input["selector"].(string)
		if selector == "" {
			return "", fmt.Errorf("click requires 'selector'")
		}
		err := chromedp.Run(ctx, 
			chromedp.WaitVisible(selector, chromedp.ByQuery),
			chromedp.Click(selector, chromedp.ByQuery),
		)
		if err != nil {
			return "", err
		}
		result = fmt.Sprintf("Clicked element: %s", selector)

	case ActionType:
		selector, _ := input["selector"].(string)
		text, _ := input["text"].(string)
		if selector == "" || text == "" {
			return "", fmt.Errorf("type requires 'selector' and 'text'")
		}
		err := chromedp.Run(ctx, 
			chromedp.WaitVisible(selector, chromedp.ByQuery),
			chromedp.SendKeys(selector, text, chromedp.ByQuery),
		)
		if err != nil {
			return "", err
		}
		result = fmt.Sprintf("Typed '%s' into %s", text, selector)

	case ActionScreenshot:
		filename, _ := input["filename"].(string)
		if filename == "" {
			filename = "screenshot.png"
		}
		var buf []byte
		err := chromedp.Run(ctx, chromedp.CaptureScreenshot(&buf))
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filename, buf, 0644); err != nil {
			return "", err
		}
		result = fmt.Sprintf("Screenshot saved to %s", filename)

	case ActionGetText:
		selector, _ := input["selector"].(string)
		if selector == "" {
			selector = "body"
		}
		var text string
		// Enhanced: Use JavaScript for cleaner text extraction of the whole page if body
		if selector == "body" {
			err := chromedp.Run(ctx, chromedp.Evaluate(`document.body.innerText`, &text))
			if err != nil {
				return "", err
			}
		} else {
			err := chromedp.Run(ctx, 
				chromedp.WaitVisible(selector, chromedp.ByQuery),
				chromedp.Text(selector, &text, chromedp.ByQuery),
			)
			if err != nil {
				return "", err
			}
		}
		result = text

	default:
		return "", fmt.Errorf("unsupported browser action: %s", action)
	}

	return result, nil
}

func (t *BrowserTool) RequiresReview() bool { return true }
