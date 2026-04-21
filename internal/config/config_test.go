package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveUsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
	p, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if p.Root != "/tmp/xdg-test/lctl" {
		t.Errorf("root mismatch: %q", p.Root)
	}
	if filepath.Base(p.Templates) != "templates" || filepath.Base(p.Scripts) != "scripts" {
		t.Errorf("subdirs wrong: %+v", p)
	}
}

func TestResolveFallsBackToHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	p, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "lctl")
	if p.Root != want {
		t.Errorf("root mismatch: got %q want %q", p.Root, want)
	}
}

func TestEnsureCreatesDirsAndSeedsTemplates(t *testing.T) {
	dir := t.TempDir()
	p := Paths{
		Root:      dir,
		Templates: filepath.Join(dir, "templates"),
		Scripts:   filepath.Join(dir, "scripts"),
	}
	if err := Ensure(p); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{p.Root, p.Templates, p.Scripts} {
		info, err := os.Stat(d)
		if err != nil {
			t.Errorf("dir missing: %s: %v", d, err)
		}
		if !info.IsDir() {
			t.Errorf("expected dir: %s", d)
		}
	}
	entries, err := os.ReadDir(p.Templates)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(defaultTemplates) {
		t.Errorf("want %d templates, got %d", len(defaultTemplates), len(entries))
	}
}

func TestEnsureDoesNotOverwriteExistingTemplates(t *testing.T) {
	dir := t.TempDir()
	p := Paths{
		Root:      dir,
		Templates: filepath.Join(dir, "templates"),
		Scripts:   filepath.Join(dir, "scripts"),
	}
	if err := Ensure(p); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(p.Templates, "blank.yaml")
	if err := os.WriteFile(target, []byte("custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Ensure(p); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "custom\n" {
		t.Errorf("existing template was overwritten")
	}
}

func TestDefaultTemplatesValidYAMLComments(t *testing.T) {
	for name, body := range defaultTemplates {
		if !strings.HasPrefix(body, "#") {
			t.Errorf("%s: should start with a comment to guide the user", name)
		}
		if !strings.Contains(body, "label:") {
			t.Errorf("%s: missing label key", name)
		}
	}
}
