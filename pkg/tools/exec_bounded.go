package tools

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// maxExecOutputBytes caps combined stdout+stderr retained in host memory for
// CLI-based executors. Container memory limits cannot constrain the host
// buffer receiving the program's output (CWE-770).
const maxExecOutputBytes = 2 << 20 // 2MiB

// boundedCombinedOutput runs cmd, capturing combined output up to the budget.
// It returns an error (killing the process) when the budget is exceeded.
func boundedCombinedOutput(cmd *exec.Cmd, budget int64) (string, error) {
	if budget <= 0 {
		budget = maxExecOutputBytes
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	var mu sync.Mutex
	remaining := budget
	overflow := false
	killed := false
	kill := func() {
		if !killed {
			killed = true
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}
	}
	var buf bytes.Buffer
	drain := func(r io.Reader) {
		tmp := make([]byte, 32*1024)
		for {
			n, rerr := r.Read(tmp)
			if n > 0 {
				mu.Lock()
				if remaining <= 0 {
					overflow = true
					kill()
					mu.Unlock()
					continue
				}
				if int64(n) > remaining {
					overflow = true
					buf.Write(tmp[:remaining])
					remaining = 0
					kill()
					mu.Unlock()
					continue
				}
				remaining -= int64(n)
				buf.Write(tmp[:n])
				mu.Unlock()
			}
			if rerr != nil {
				return
			}
		}
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); drain(stdout) }()
	go func() { defer wg.Done(); drain(stderr) }()
	wg.Wait()
	err = cmd.Wait()
	if overflow {
		return "", fmt.Errorf("execution output exceeded %d byte budget (truncated)", budget)
	}
	return buf.String(), err
}
