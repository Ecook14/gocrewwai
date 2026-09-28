package delegation

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Ecook14/gocrewwai/pkg/core"
)

// MaxDelegationDepth bounds recursive A2A delegation chains. It enforces the
// collaboration.md max-depth contract and prevents infinite coworker loops.
const MaxDelegationDepth = 5

// depthKey carries the current delegation depth in a context.
type depthKey struct{}

func delegationDepth(ctx context.Context) int {
	if v, ok := ctx.Value(depthKey{}).(int); ok {
		return v
	}
	return 0
}

// withDepth returns a context with incremented delegation depth.
func withDepth(ctx context.Context) context.Context {
	return context.WithValue(ctx, depthKey{}, delegationDepth(ctx)+1)
}

var bridgeMu sync.RWMutex

// registeredAdapters maps agent role -> core.Agent for A2A bridging.
var registeredAdapters = make(map[string]core.Agent)

// RegisterA2AAdapter registers a remote agent adapter for A2A delegation.
func RegisterA2AAdapter(role string, agent core.Agent) {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	registeredAdapters[strings.ToLower(strings.TrimSpace(role))] = agent
}

// DelegateViaA2A delegates a task to a registered remote agent by role.
// It enforces MaxDelegationDepth and requires non-empty task text (DoS guard).
func DelegateViaA2A(ctx context.Context, role, task, taskContext string) (string, error) {
	if strings.TrimSpace(role) == "" {
		return "", fmt.Errorf("delegation: coworker role is required")
	}
	if strings.TrimSpace(task) == "" {
		return "", fmt.Errorf("delegation: task is required")
	}
	if len(task) > 20000 {
		return "", fmt.Errorf("delegation: task exceeds 20000 character limit")
	}
	depth := delegationDepth(ctx)
	if depth >= MaxDelegationDepth {
		return "", fmt.Errorf("delegation: max depth %d exceeded (cycle guard)", MaxDelegationDepth)
	}
	bridgeMu.RLock()
	target, ok := registeredAdapters[strings.ToLower(strings.TrimSpace(role))]
	bridgeMu.RUnlock()
	if !ok {
		return "", fmt.Errorf("delegation: no A2A adapter registered for role %q", role)
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	res, err := target.Execute(withDepth(ctx), task+". Context: "+taskContext, map[string]interface{}{"delegation_depth": depth + 1})
	if err != nil {
		return "", fmt.Errorf("delegation: remote execute failed: %w", err)
	}
	return fmt.Sprintf("%v", res), nil
}
