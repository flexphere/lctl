package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureSeedsConfigOnce(t *testing.T) {
	dir := t.TempDir()
	p := Paths{
		Root:      dir,
		Templates: filepath.Join(dir, "templates"),
		Scripts:   filepath.Join(dir, "scripts"),
	}
	if err := Ensure(p); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[list]") {
		t.Errorf("seeded config missing [list]: %s", string(data))
	}
	// Second Ensure must not overwrite user edits.
	if err := os.WriteFile(path, []byte("# user-edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Ensure(p); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	if string(data) != "# user-edit\n" {
		t.Errorf("user edits overwritten: %s", string(data))
	}
}

func TestSeededConfigParses(t *testing.T) {
	dir := t.TempDir()
	p := Paths{
		Root:      dir,
		Templates: filepath.Join(dir, "templates"),
		Scripts:   filepath.Join(dir, "scripts"),
	}
	if err := Ensure(p); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSettings(p)
	if err != nil {
		t.Fatalf("seeded config must parse: %v", err)
	}
	// All keys commented out → defaults should still apply.
	if s.Keymap.New != "n" {
		t.Errorf("seeded defaults should yield defaults: %+v", s.Keymap)
	}
}
