package serve

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Subprocess manages a background dev tool process (e.g., Vite).
type Subprocess struct {
	cmd *exec.Cmd
}

// StartSubprocess spawns the given shell command as a background process,
// forwarding its stdout/stderr. Returns nil if command is empty.
func StartSubprocess(command string) (*Subprocess, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, nil
	}

	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting dev command %q: %w", command, err)
	}

	return &Subprocess{cmd: cmd}, nil
}

// Stop sends a signal to terminate the subprocess and waits for it.
func (s *Subprocess) Stop() {
	if s == nil || s.cmd == nil || s.cmd.Process == nil {
		return
	}
	s.cmd.Process.Signal(os.Interrupt)
	s.cmd.Wait()
}
