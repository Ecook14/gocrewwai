package llm

// The registry logic has been moved to pkg/config to break an import cycle.
// Use config.GetClient(modelName) instead of llm.GetClient(modelName).
