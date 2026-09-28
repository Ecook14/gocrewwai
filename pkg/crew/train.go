package crew

import (
	"context"
	"fmt"
	"time"
)

// RecordFeedback captures human feedback for a completed crew run as a
// TrainingExample so future runs improve. The canonical crew training loop
// lives in crew.go (Crew.Train / ProvideTrainingFeedback); this helper is the
// package-level entry point for single feedback records backed by the
// TrainingDataset type in training_data.go.
func RecordFeedback(ctx context.Context, crewID string, feedback string) error {
	if crewID == "" {
		return fmt.Errorf("crew: RecordFeedback requires a crew ID")
	}
	if feedback == "" {
		return fmt.Errorf("crew: RecordFeedback requires feedback text")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	ds := NewTrainingDataset("crew-" + crewID)
	ds.AddExample(TrainingExample{
		TaskDescription: "crew " + crewID + " feedback",
		Feedback:        feedback,
		Timestamp:       time.Now().Format("2006-01-02T15:04:05Z07:00"),
	})
	return nil
}
