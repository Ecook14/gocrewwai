package tasks

import (
	"sync"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/i18n"
)

// TestTask_ConcurrentSnapshot runs Execute while readers snapshot state.
// Run with -race: any unlocked access fails the suite.
func TestTask_ConcurrentSnapshot(t *testing.T) {
	mkTask := func() *Task {
		tk := NewTask("do work", &fakeAgent{role: "r"})
		tk.I18N, _ = i18n.NewI18N("en")
		return tk
	}
	task := mkTask()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				_, _, _, _ = task.Snapshot()
				_ = task.IsProcessed()
				_ = task.IsFailed()
				_ = task.GetError()
				_ = task.GetOutput()
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 25; j++ {
			tk := mkTask()
			_ = tk
			task.SetOutput(j)
			task.SetProcessed(j%2 == 0)
			task.SetFailed(false)
			task.SetError(nil)
		}
	}()
	wg.Wait()
}
