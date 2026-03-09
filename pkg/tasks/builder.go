package tasks

import (
	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

// TaskBuilder provides a fluent API for constructing Tasks.
type TaskBuilder struct {
	task *Task
}

func NewTaskBuilder() *TaskBuilder {
	return &TaskBuilder{
		task: &Task{
			MaxRetries: 3,
			MaxCycles:  5,
		},
	}
}

func (b *TaskBuilder) Description(desc string) *TaskBuilder {
	b.task.Description = desc
	return b
}

func (b *TaskBuilder) ExpectedOutput(expected string) *TaskBuilder {
	b.task.ExpectedOutput = expected
	return b
}

func (b *TaskBuilder) Agent(a *agents.Agent) *TaskBuilder {
	b.task.Agent = a
	return b
}

func (b *TaskBuilder) Tools(t ...tools.Tool) *TaskBuilder {
	b.task.Tools = append(b.task.Tools, t...)
	return b
}

func (b *TaskBuilder) AsyncExecution(v bool) *TaskBuilder {
	b.task.AsyncExecution = v
	return b
}

func (b *TaskBuilder) OutputSchema(schema string) *TaskBuilder {
	b.task.OutputSchema = schema
	return b
}

func (b *TaskBuilder) HumanInput(v bool) *TaskBuilder {
	b.task.HumanInput = v
	return b
}

func (b *TaskBuilder) Context(tasks ...*Task) *TaskBuilder {
	b.task.Context = append(b.task.Context, tasks...)
	return b
}

func (b *TaskBuilder) OutputFile(path string) *TaskBuilder {
	b.task.OutputFile = path
	return b
}

func (b *TaskBuilder) Build() *Task {
	return b.task
}
