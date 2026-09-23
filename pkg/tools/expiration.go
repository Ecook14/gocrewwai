// Auto-generated expiration constants for Gocrewwai caching.
// These durations define how long cached tool results remain valid
// before stale data could mislead agents. Adjust per tool sensitivity.

package tools

const (
	// DefaultCacheTTL is the fallback expiration for cached tool outputs
	// when a tool does not specify its own TTL.
	DefaultCacheTTL = 300 * 1000 * 1000 * 1000 // 5 minutes in nanoseconds

	// ShortCacheTTL is for fast-changing data (search results, live prices).
	ShortCacheTTL = 30 * 1000 * 1000 * 1000 // 30 seconds

	// LongCacheTTL is for stable reference data (config, schema lookups).
	LongCacheTTL = 3600 * 1000 * 1000 * 1000 // 1 hour
)
