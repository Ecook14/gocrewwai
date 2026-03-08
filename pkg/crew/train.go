package crew

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Train runs the crew in an interactive mode where the user can provide 
// feedback for each task. These "corrections" are saved as few-shot examples 
// in the agent's memory/prompt for future runs.
func (c *Crew) Train(ctx context.Context, iterations int) error {
	if c.Verbose {
		defaultLogger.Info("🎓 Starting Training Mode", slog.Int("iterations", iterations))
	}

	reader := bufio.NewReader(os.Stdin)

	for i := 0; i < iterations; i++ {
		defaultLogger.Info("--- 🎓 Training Iteration ---", slog.Int("iter", i+1), slog.Int("total", iterations))
		
		for _, task := range c.Tasks {
			if c.Verbose {
				defaultLogger.Info("Executing Training Task", "description", task.Description)
			}

			// Execute task
			result, err := task.Execute(ctx)
			if err != nil {
				return fmt.Errorf("task failed during training: %w", err)
			}

			defaultLogger.Info("[🤖 Agent Output]", slog.String("role", task.Agent.Role), slog.Any("result", result))
			fmt.Print("\n[🎓 Feedback] Provide a better version or feedback (press Enter to accept as-is): ")
			
			feedback, _ := reader.ReadString('\n')
			feedback = strings.TrimSpace(feedback)

			if feedback != "" {
				example := fmt.Sprintf("Input: %s\nIdeal Output: %s", task.Description, feedback)
				task.Agent.FewShotExamples = append(task.Agent.FewShotExamples, example)
				fmt.Println("✅ Feedback saved as few-shot example for the agent.")
			} else {
				fmt.Println("✅ Output accepted.")
			}
		}

		// Reset tasks for next iteration
		for _, t := range c.Tasks {
			t.Processed = false
			t.Output = nil
		}
	}

	if c.Verbose {
		defaultLogger.Info("🎓 Training Complete. Agent prompts are now optimized.")
	}

	return nil
}
