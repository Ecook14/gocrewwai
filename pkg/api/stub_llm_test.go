package api

import (
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/testutil"
)

// stubLLMClient swaps the kickoff provider constructor for an in-memory stub
// and returns the mock (for asserting on recorded prompts) plus a restore
// func. Tests that drive an accepted (202) kickoff must use this: the crew
// goroutine would otherwise dial the real provider endpoint, receive a 401,
// and retry, which is slow, flaky, and sends outbound traffic from CI.
func stubLLMClient(t *testing.T) (*testutil.MockClient, func()) {
	t.Helper()
	prev := newLLMClient
	mock := testutil.NewSimpleMock("stubbed completion")
	newLLMClient = func(string) llm.Client {
		return mock
	}
	return mock, func() { newLLMClient = prev }
}
