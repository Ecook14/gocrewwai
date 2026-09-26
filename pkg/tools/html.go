package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Ecook14/gocrewwai/pkg/utils"
)

// HTMLReadTool allows agents to extract readable text from HTML files.
type HTMLReadTool struct {
	BaseTool
	Chroot string
}

var _ Tool = (*HTMLReadTool)(nil)

func NewHTMLReadTool(chroot string) *HTMLReadTool {
	return &HTMLReadTool{
		BaseTool: BaseTool{
			NameValue:        "HTMLReadTool",
			DescriptionValue: "Reads and extracts text content from an HTML file, stripping tags. Input requires 'file_path' as a string.",
		},
		Chroot: chroot,
	}
}

func (t *HTMLReadTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	pathRaw, ok := input["file_path"]
	if !ok {
		return "", fmt.Errorf("missing 'file_path' in input")
	}
	path, ok := pathRaw.(string)
	if !ok {
		return "", fmt.Errorf("'file_path' must be a string")
	}

	safePath, err := utils.ValidatePath(path, t.Chroot)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return stripHTMLTagsOnly(string(content)), nil
}

func (t *HTMLReadTool) Name() string        { return t.BaseTool.NameValue }
func (t *HTMLReadTool) Description() string { return t.BaseTool.DescriptionValue }

// CacheFunction isolates cache entries per file path.
func (t *HTMLReadTool) CacheFunction(input map[string]interface{}) string {
	if p, ok := input["file_path"].(string); ok && p != "" {
		return "HTMLReadTool:" + p
	}
	return ""
}

// stripHTML removes HTML tags and decodes basic entities from content.
// Whitespace inside <pre>...</pre> blocks is preserved; outside <pre>,
// runs of whitespace are collapsed to a single space.
func stripHTMLTagsOnly(html string) string {
	// Remove script and style blocks first (their content is never visible).
	html = removeTag(html, "script")
	html = removeTag(html, "style")

	var result strings.Builder
	var inTag bool
	var inPre bool
	var lastChar byte
	var tagName strings.Builder // accumulates the current tag name for closing-tag detection

	for i := 0; i < len(html); i++ {
		ch := html[i]

		if !inTag && ch == '<' {
			// Opening delimiter — capture tag name for later closing-tag check.
			tagName.Reset()
			inTag = true
			continue
		}

		if inTag {
			if ch == '>' {
				inTag = false
				// Determine if this was an opening or closing tag from the captured name.
				tn := strings.ToLower(tagName.String())
				if tn == "/pre" {
					inPre = false
				} else if tn == "pre" {
					inPre = true
				}
			} else if ch != ' ' && ch != '	' && ch != '\n' && ch != '\r' {
				// Accumulate tag name characters (skip whitespace inside tag).
				tagName.WriteByte(ch)
			}
			continue
		}

		if inPre {
			// Inside <pre>: preserve every character verbatim, including whitespace.
			result.WriteByte(ch)
			lastChar = ch
		} else {
			// Outside <pre>: collapse runs of whitespace to a single space.
			if ch == ' ' || ch == '	' || ch == '\n' || ch == '\r' {
				if lastChar != ' ' {
					result.WriteByte(' ')
					lastChar = ' '
				}
			} else {
				result.WriteByte(ch)
				lastChar = ch
			}
		}
	}

	// Decode common HTML entities (after whitespace handling so entities
	// inside <pre> are also decoded). Single-pass replacer avoids O(n*k)
	// allocations from sequential ReplaceAll calls.
	text := htmlEntityReplacer.Replace(result.String())

	return text
}

// htmlEntityReplacer decodes common entities in a single pass.
var htmlEntityReplacer = strings.NewReplacer(
	"&nbsp;", " ",
	"&amp;", "&",
	"&lt;", "<",
	"&gt;", ">",
	"&quot;", "\"",
	"&#39;", "'",
	"&#x27;", "'",
)

func removeTag(s, tag string) string {
	open := "<" + tag
	close := "</" + tag + ">"
	for {
		start := strings.Index(s, open)
		if start == -1 {
			return s
		}
		end := strings.Index(s[start:], close)
		if end == -1 {
			return s[:start]
		}
		s = s[:start] + s[start+end+len(close):]
	}
}

// stripHTML removes HTML tags and decodes basic entities from content.
