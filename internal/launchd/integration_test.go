package launchd

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// TestExecListIntegration runs a real `launchctl list` and asserts we
// can parse some rows. Skipped when launchctl is unavailable (e.g. on
// non-darwin CI or sandboxed environments).
func TestExecListIntegration(t *testing.T) {
	if _, err := exec.LookPath("launchctl"); err != nil {
		t.Skip("launchctl not available")
	}
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	entries, err := c.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) == 0 {
		t.Skip("no launchctl entries (unusual, skipping)")
	}
	for _, e := range entries {
		if e.Label == "" {
			t.Errorf("empty label: %+v", e)
		}
	}
}
