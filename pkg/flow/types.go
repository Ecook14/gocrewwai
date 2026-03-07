package flow

import "context"

// FlowTrackable represents a unit of orchestratable task.
type FlowTrackable interface {
	GetID() string
	Status() string
	Metadata() map[string]interface{}
}

// StreamChunk provides async iterator streaming data type translations.
type StreamChunk struct {
	Content      string
	IsFinalBlock bool
	ToolCall     *ToolCallData
}

// ToolCallData defines details captured when a stream requests a tool
type ToolCallData struct {
	Name      string
	Arguments map[string]interface{}
}

// AsyncChunkGenerator creates a pattern mapping the `async def aexecute()` pattern
type AsyncChunkGenerator interface {
	Next(ctx context.Context) (*StreamChunk, error)
}
