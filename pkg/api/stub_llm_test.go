package api

import (
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/testutil"
)

// stubLLMClient swaps the kickoff provider constructor for an in-memory stub
// and returns a restore func. Tests that drive an accepted (202) kickoff must
// use this: the crew goroutine would otherwise dial the real provider endpoint,
// receive a 401, and retry, which is slow, flaky, and sends outbound traffic
// from CI.
func stubLLMClient(t *testing.T) (restore func()) {
	t.Helper()
	prev := newLLMClient
	newLLMClient = func(string) llm.Client {
		return testutil.NewSimpleMock("stubbed completion")
	}
	return func() { newLLMClient = prev }
}
