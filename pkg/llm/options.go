package llm

// Options provides a standardized, provider-agnostic configuration struct for LLM calls.
// This mirrors the full parameter set from CrewAI Python's LLM configuration.
// These options are passed via the `options` map in the Client interface, but this
// struct provides typed access and documentation.
type Options struct {
	// Model identifier (e.g., "gpt-4o", "gemini-2.0-flash")
	Model string `json:"model,omitempty"`

	// Temperature controls randomness (0.0–1.0)
	Temperature *float64 `json:"temperature,omitempty"`

	// Timeout is the max wait time for response (seconds)
	Timeout int `json:"timeout,omitempty"`

	// MaxTokens is the maximum response length
	MaxTokens int `json:"max_tokens,omitempty"`

	// TopP is the nucleus sampling parameter (0.0–1.0)
	TopP *float64 `json:"top_p,omitempty"`

	// FrequencyPenalty reduces word repetition (-2.0 to 2.0)
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// PresencePenalty encourages topic diversity (-2.0 to 2.0)
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// ResponseFormat specifies output structure (e.g., {"type": "json_object"})
	ResponseFormat map[string]interface{} `json:"response_format,omitempty"`

	// Seed for reproducible results
	Seed *int `json:"seed,omitempty"`

	// APIKey overrides the provider API key
	APIKey string `json:"api_key,omitempty"`

	// BaseURL overrides the custom API endpoint
	BaseURL string `json:"base_url,omitempty"`

	// Stop sequences to halt generation
	Stop []string `json:"stop,omitempty"`

	// Stream enables streaming output
	Stream bool `json:"stream,omitempty"`

	// MaxRetries is the maximum retry attempts
	MaxRetries int `json:"max_retries,omitempty"`

	// Logprobs returns log probabilities
	Logprobs bool `json:"logprobs,omitempty"`

	// TopLogprobs is the number of most likely tokens to return
	TopLogprobs int `json:"top_logprobs,omitempty"`

	// ReasoningEffort for o1/o3 models: "low", "medium", "high"
	ReasoningEffort string `json:"reasoning_effort,omitempty"`

	// MaxCompletionTokens for newer models (o1, etc.)
	MaxCompletionTokens int `json:"max_completion_tokens,omitempty"`

	// Organization ID
	Organization string `json:"organization,omitempty"`

	// Project ID
	Project string `json:"project,omitempty"`
}

// ToMap converts the typed Options struct into the map[string]interface{} format
// expected by the Client interface. Only non-zero values are included.
func (o *Options) ToMap() map[string]interface{} {
	m := make(map[string]interface{})

	if o.Model != "" {
		m["model"] = o.Model
	}
	if o.Temperature != nil {
		m["temperature"] = *o.Temperature
	}
	if o.Timeout > 0 {
		m["timeout"] = o.Timeout
	}
	if o.MaxTokens > 0 {
		m["max_tokens"] = o.MaxTokens
	}
	if o.TopP != nil {
		m["top_p"] = *o.TopP
	}
	if o.FrequencyPenalty != nil {
		m["frequency_penalty"] = *o.FrequencyPenalty
	}
	if o.PresencePenalty != nil {
		m["presence_penalty"] = *o.PresencePenalty
	}
	if o.ResponseFormat != nil {
		m["response_format"] = o.ResponseFormat
	}
	if o.Seed != nil {
		m["seed"] = *o.Seed
	}
	if len(o.Stop) > 0 {
		m["stop"] = o.Stop
	}
	if o.Stream {
		m["stream"] = true
	}
	if o.MaxRetries > 0 {
		m["max_retries"] = o.MaxRetries
	}
	if o.Logprobs {
		m["logprobs"] = true
	}
	if o.TopLogprobs > 0 {
		m["top_logprobs"] = o.TopLogprobs
	}
	if o.ReasoningEffort != "" {
		m["reasoning_effort"] = o.ReasoningEffort
	}
	if o.MaxCompletionTokens > 0 {
		m["max_completion_tokens"] = o.MaxCompletionTokens
	}
	if o.Organization != "" {
		m["organization"] = o.Organization
	}
	if o.Project != "" {
		m["project"] = o.Project
	}

	return m
}

// Float64 is a helper to create a *float64 for use in Options.
func Float64(v float64) *float64 { return &v }

// Int is a helper to create a *int for use in Options.
func Int(v int) *int { return &v }
