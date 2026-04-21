package config

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// captureWarnings swaps warnWriter for the duration of fn so tests
// can assert on the exact warnings LoadSettings emits. The previous
// writer is restored when fn returns, even on panic.
func captureWarnings(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	prev := warnWriter
	warnWriter = &buf
	defer func() { warnWriter = prev }()
	fn()
	return buf.String()
}

func TestLoadSettingsWarnsOnUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	body := `[list]
auto_refresh = true

[keyamp]                    # typo: should be "keymap"
new = "N"

[list.status]
runnnig = "42"              # typo: should be "running"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var got Settings
	stderr := captureWarnings(t, func() {
		var err error
		got, err = LoadSettings(Paths{Root: dir})
		if err != nil {
			t.Fatal(err)
		}
	})

	// Valid key still applied.
	if !got.List.AutoRefresh {
		t.Errorf("known key auto_refresh should still apply, got %v", got.List.AutoRefresh)
	}
	// Both typos should have surfaced.
	for _, want := range []string{"keyamp.new", "list.status.runnnig"} {
		if !bytes.Contains([]byte(stderr), []byte(want)) {
			t.Errorf("expected warning to mention %q, got:\n%s", want, stderr)
		}
	}
}

func TestLoadSettingsNoWarningsForCleanConfig(t *testing.T) {
	dir := t.TempDir()
	body := `[list]
auto_refresh = true

[keymap]
new = "N"

[list.status]
running = "42"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	stderr := captureWarnings(t, func() {
		if _, err := LoadSettings(Paths{Root: dir}); err != nil {
			t.Fatal(err)
		}
	})
	if stderr != "" {
		t.Errorf("clean config should produce no warnings, got:\n%s", stderr)
	}
}

// TestSeedMatchesDefaults guards against the seeded config.toml
// drifting away from DefaultSettings. The seed ships every option
// commented out, so loading it must produce exactly the compiled-in
// defaults — any mismatch means either the seed or DefaultSettings
// was updated without the other.
func TestSeedMatchesDefaults(t *testing.T) {
	dir := t.TempDir()
	p := Paths{
		Root:      dir,
		Templates: filepath.Join(dir, "templates"),
		Scripts:   filepath.Join(dir, "scripts"),
	}
	if err := Ensure(p); err != nil {
		t.Fatal(err)
	}
	// Sanity check — the seed was actually written.
	seedPath := filepath.Join(dir, "config.toml")
	if _, err := os.Stat(seedPath); err != nil {
		t.Fatalf("seed not written: %v", err)
	}
	// Also catch any unknown-key drift by failing if the seed triggers
	// warnings (which it shouldn't; every line is a comment).
	stderr := captureWarnings(t, func() {
		got, err := LoadSettings(p)
		if err != nil {
			t.Fatal(err)
		}
		want := DefaultSettings()
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("seed/DefaultSettings drift:\n got=%#v\nwant=%#v", got, want)
		}
	})
	if stderr != "" {
		t.Errorf("seeded config should not emit warnings, got:\n%s", stderr)
	}
}

// silence the io import warning when tests compile without using io
// directly (captureWarnings returns string but we kept io.Writer in
// the variable type for future flexibility).
var _ io.Writer = warnWriter
