package tools

import (
	"context"
	"fmt"
)

// CalculatorTool provides basic mathematical operations.
type CalculatorTool struct {
	BaseTool
}

func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{
		BaseTool: BaseTool{
			NameValue: "Calculator",
			DescriptionValue: "Perform basic math operations. Input requires 'expression' (e.g. '2 + 2'). " +
				"Supports addition (+), subtraction (-), multiplication (*), and division (/).",
		},
	}
}


func (t *CalculatorTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	exprRaw, ok := input["expression"]
	if !ok {
		return "", fmt.Errorf("missing 'expression' in input")
	}
	expr, ok := exprRaw.(string)
	if !ok {
		return "", fmt.Errorf("'expression' must be a string")
	}

	// Advanced Level: Using a simple evaluator for basic math
	result, err := evaluateExpression(expr)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate expression '%s': %w", expr, err)
	}

	return fmt.Sprintf("%v", result), nil
}

func (t *CalculatorTool) RequiresReview() bool { return false }

// evaluateExpression is a simple helper for the advanced level demo.
// In a real production tool, we'd use a full AST parser like go-lua or gopher-lua
// or a specialized math expression evaluator.
func evaluateExpression(expr string) (float64, error) {
	// For now, let's support basic binary expressions "x op y"
	// or just return a mock if it's too complex for this demo.
	// But let's try a simple split for the demo.
	var a, b float64
	var op string
	n, err := fmt.Sscanf(expr, "%f %s %f", &a, &op, &b)
	if err != nil || n < 3 {
		return 0, fmt.Errorf("unsupported expression format. Use 'x op y' (e.g. '10 + 5'). Important: Ensure spaces between numbers and the operator.")
	}

	switch op {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	case "/":
		if b == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return a / b, nil
	default:
		return 0, fmt.Errorf("unsupported operator: %s", op)
	}
}
