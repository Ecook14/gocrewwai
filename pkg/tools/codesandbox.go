package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type CodeSandboxTool struct {
	BaseTool
	StorageDir string
}

func NewCodeSandboxTool() *CodeSandboxTool {
	return &CodeSandboxTool{
		BaseTool: BaseTool{
			NameValue:        "CodeSandboxTool",
			DescriptionValue: "Executes Python or JavaScript code safely. Specify 'language' (python/javascript) and 'code'.",
		},
		StorageDir: os.TempDir(),
	}
}

func (t *CodeSandboxTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	lang, _ := input["language"].(string)
	code, _ := input["code"].(string)

	if code == "" {
		return "", fmt.Errorf("missing 'code' parameter")
	}

	var cmd *exec.Cmd
	switch lang {
	case "python":
		cmd = exec.CommandContext(ctx, "python3", "-c", code)
	case "javascript", "js":
		cmd = exec.CommandContext(ctx, "node", "-e", code)
	default:
		return "", fmt.Errorf("unsupported language: %s", lang)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("execution error: %w", err)
	}

	return string(output), nil
}
