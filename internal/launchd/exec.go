package launchd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Exec is the production Client that shells out to the `launchctl`
// binary. It is safe for concurrent use by multiple goroutines.
type Exec struct {
	// Bin is the path to launchctl; defaults to "launchctl" (via PATH)
	// when empty.
	Bin string
	// UID is the launch session user identifier, typically
	// current process uid. Lazily resolved when zero.
	UID int
}

// New constructs an Exec client with defaults, resolving the current
// user's uid.
func New() (*Exec, error) {
	uid, err := currentUID()
	if err != nil {
		return nil, err
	}
	if _, err := exec.LookPath("launchctl"); err != nil {
		return nil, ErrNotInstalled
	}
	return &Exec{Bin: "launchctl", UID: uid}, nil
}

func (e *Exec) bin() string {
	if e.Bin == "" {
		return "launchctl"
	}
	return e.Bin
}

func (e *Exec) target(label string) string { return Target(e.UID, label) }
func (e *Exec) domain() string             { return Domain(e.UID) }

// List runs `launchctl list` and parses its output.
func (e *Exec) List(ctx context.Context) ([]ListEntry, error) {
	out, err := e.run(ctx, "list")
	if err != nil {
		return nil, err
	}
	return parseList(bytes.NewReader(out))
}

// PrintDisabled returns the administrative enable/disable map for the
// current user domain. Keys are labels; value is true when disabled.
// Labels not present in the result are considered enabled.
func (e *Exec) PrintDisabled(ctx context.Context) (map[string]bool, error) {
	out, err := e.run(ctx, "print-disabled", e.domain())
	if err != nil {
		return nil, err
	}
	return parsePrintDisabled(bytes.NewReader(out))
}

// Bootstrap registers the plist file into the current user's domain.
// Equivalent to `launchctl bootstrap gui/<uid> <path>`.
func (e *Exec) Bootstrap(ctx context.Context, plistPath string) error {
	if plistPath == "" {
		return errors.New("plist path required")
	}
	_, err := e.run(ctx, "bootstrap", e.domain(), plistPath)
	return err
}

// Bootout unregisters the service. Missing services are treated as
// non-fatal ("service not found" errors are swallowed).
func (e *Exec) Bootout(ctx context.Context, label string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	_, err := e.run(ctx, "bootout", e.target(label))
	return ignoreNotFound(err)
}

// Kickstart triggers an immediate run with -k (kill any existing
// instance first).
func (e *Exec) Kickstart(ctx context.Context, label string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	_, err := e.run(ctx, "kickstart", "-k", e.target(label))
	return err
}

// Kill sends a signal to the running service instance.
func (e *Exec) Kill(ctx context.Context, label, signal string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	if signal == "" {
		signal = "TERM"
	}
	_, err := e.run(ctx, "kill", signal, e.target(label))
	return err
}

// Enable marks a service as enabled in the domain.
func (e *Exec) Enable(ctx context.Context, label string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	_, err := e.run(ctx, "enable", e.target(label))
	return err
}

// Disable marks a service as disabled in the domain.
func (e *Exec) Disable(ctx context.Context, label string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	_, err := e.run(ctx, "disable", e.target(label))
	return err
}

func (e *Exec) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, e.bin(), args...) //nolint:gosec // args are validated
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("launchctl %s failed: %w: %s",
			strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// validateLabel rejects labels that would reach shell-sensitive
// characters or target escape attempts.
func validateLabel(label string) error {
	if label == "" {
		return errors.New("label required")
	}
	for _, r := range label {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
			continue
		default:
			return fmt.Errorf("invalid character %q in label", r)
		}
	}
	return nil
}

// ignoreNotFound turns launchctl's "service not found" error into nil,
// so reload flows can be idempotent.
func ignoreNotFound(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "No such process") ||
		strings.Contains(msg, "Could not find service") ||
		strings.Contains(msg, "Bootout failed: 3") {
		return nil
	}
	return err
}
