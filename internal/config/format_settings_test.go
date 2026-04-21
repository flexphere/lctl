package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLayoutAndVariables(t *testing.T) {
	dir := t.TempDir()
	body := `[list.layout]
template = "$cursor$label"
divider  = " | "

[list.variables.label]
text  = "Label: $value"
color = "#ffffff"
width = 60

[list.variables.state]
color = "220"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSettings(Paths{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if s.List.Layout.Template != "$cursor$label" {
		t.Errorf("template override lost: %q", s.List.Layout.Template)
	}
	if s.List.Layout.Divider != " | " {
		t.Errorf("divider override lost: %q", s.List.Layout.Divider)
	}
	label := s.List.Variables["label"]
	if label.Text != "Label: $value" {
		t.Errorf("variables.label.text override lost: %q", label.Text)
	}
	if label.Color != "#ffffff" {
		t.Errorf("variables.label.color override lost: %q", label.Color)
	}
	if label.Width != 60 {
		t.Errorf("variables.label.width override lost: %d", label.Width)
	}
	// Partially-overridden state: color set, text/width inherited.
	state := s.List.Variables["state"]
	if state.Color != "220" {
		t.Errorf("variables.state.color override lost: %q", state.Color)
	}
	if state.Text != "$value" {
		t.Errorf("variables.state.text should inherit default, got %q", state.Text)
	}
	if state.Width != 14 {
		t.Errorf("variables.state.width should inherit default 14, got %d", state.Width)
	}
	// Untouched variable falls back entirely to default.
	enabled := s.List.Variables["enabled"]
	if enabled.Text != "$value" || enabled.Width != 8 {
		t.Errorf("variables.enabled default not applied: %+v", enabled)
	}
}

func TestLoadLayoutUsesDefaultsWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSettings(Paths{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if s.List.Layout.Template == "" {
		t.Error("template default should be populated")
	}
	if s.List.Layout.Divider == "" {
		t.Error("divider default should be populated")
	}
	for _, name := range []string{"label", "state", "exit", "next_run", "schedule", "enabled"} {
		v := s.List.Variables[name]
		if v.Width <= 0 {
			t.Errorf("variables.%s.width default should be positive, got %d", name, v.Width)
		}
		if v.Text == "" {
			t.Errorf("variables.%s.text default should be populated", name)
		}
	}
}

func TestLoadLayoutZeroWidthFallsBack(t *testing.T) {
	dir := t.TempDir()
	body := `[list.variables.label]
width = 0
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSettings(Paths{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if s.List.Variables["label"].Width != 40 {
		t.Errorf("zero width should fall back to default, got %d", s.List.Variables["label"].Width)
	}
}
