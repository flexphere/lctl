// Package editor resolves the user's preferred terminal editor and
// builds exec.Cmd values that can be passed to tea.ExecProcess.
package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Resolve returns the terminal editor command to use, following the
// convention: $VISUAL > $EDITOR > vi.
//
// The returned slice is already split on whitespace so callers can
// pass it straight to exec.Command via [0], [1:]...
func Resolve() []string {
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return strings.Fields(v)
		}
	}
	return []string{"vi"}
}

// Command builds an exec.Cmd that runs the resolved editor on path.
// Stdio is left nil so the caller (e.g. tea.ExecProcess) can wire it
// to the real terminal.
func Command(path string) (*exec.Cmd, error) {
	if path == "" {
		return nil, fmt.Errorf("editor: path required")
	}
	parts := Resolve()
	args := append(parts[1:], path) //nolint:gocritic // intentional: editor args + file
	return exec.Command(parts[0], args...), nil
}
