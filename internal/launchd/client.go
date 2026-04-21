// Package launchd wraps the `launchctl` binary so the TUI can perform
// job operations without issuing shell strings directly.
package launchd

import (
	"context"
	"errors"
	"fmt"
	"os/user"
	"strconv"
)

// ListEntry is a row from `launchctl list`.
type ListEntry struct {
	Label    string
	PID      *int // nil when not running
	LastExit *int // nil when unknown
}

// Running reports whether the agent appears to have a live process.
func (e ListEntry) Running() bool { return e.PID != nil }

// State classifies the runtime state of a job into a short string
// suitable for the Dashboard.
type State int

const (
	StateUnknown State = iota
	StateLoaded
	StateRunning
	StateErrored
)

// String returns a short stable label for the state.
func (s State) String() string {
	switch s {
	case StateLoaded:
		return "loaded"
	case StateRunning:
		return "running"
	case StateErrored:
		return "errored"
	default:
		return "unknown"
	}
}

// StateOf computes the Dashboard state from a ListEntry.
func StateOf(e ListEntry) State {
	if e.PID != nil {
		return StateRunning
	}
	if e.LastExit != nil && *e.LastExit != 0 {
		return StateErrored
	}
	return StateLoaded
}

// Client defines the surface used by the TUI.
type Client interface {
	List(ctx context.Context) ([]ListEntry, error)
	PrintDisabled(ctx context.Context) (map[string]bool, error)
	Bootstrap(ctx context.Context, plistPath string) error
	Bootout(ctx context.Context, label string) error
	Kickstart(ctx context.Context, label string) error
	Kill(ctx context.Context, label, signal string) error
	Enable(ctx context.Context, label string) error
	Disable(ctx context.Context, label string) error
}

// Target returns the modern domain-qualified target for a user agent,
// e.g. "gui/501/com.flexphere.job".
func Target(uid int, label string) string {
	return fmt.Sprintf("gui/%d/%s", uid, label)
}

// Domain returns the user domain, e.g. "gui/501".
func Domain(uid int) string {
	return fmt.Sprintf("gui/%d", uid)
}

// currentUID reports the current user's numeric uid, used to compute
// the launchctl GUI domain.
func currentUID() (int, error) {
	u, err := user.Current()
	if err != nil {
		return 0, fmt.Errorf("lookup user: %w", err)
	}
	n, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, fmt.Errorf("parse uid: %w", err)
	}
	return n, nil
}

// ErrNotInstalled signals that launchctl is absent from PATH.
var ErrNotInstalled = errors.New("launchctl not found in PATH")
