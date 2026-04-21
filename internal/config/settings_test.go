package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	if s.Keymap.New == "" || s.List.Brand.Color == "" {
		t.Error("defaults should be fully populated")
	}
	if !s.List.AutoRefresh {
		t.Error("auto refresh default should be on")
	}
	if s.List.RefreshInterval <= 0 {
		t.Error("refresh interval default must be positive")
	}
}

func TestLoadSettingsMissingFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	p := Paths{Root: dir}
	s, err := LoadSettings(p)
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if s.Keymap.New != "n" {
		t.Errorf("default keymap not applied: %+v", s.Keymap)
	}
}

func TestLoadSettingsMergesOverDefaults(t *testing.T) {
	dir := t.TempDir()
	body := `[list]
auto_refresh = true
refresh_interval = 5

[keymap]
new = "N"

[list.brand]
color = "#ff0000"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSettings(Paths{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !s.List.AutoRefresh {
		t.Error("auto_refresh not read")
	}
	if s.List.RefreshInterval != 5 {
		t.Errorf("interval not 5: %v", s.List.RefreshInterval)
	}
	if s.Keymap.New != "N" {
		t.Errorf("keymap override lost: %q", s.Keymap.New)
	}
	if s.Keymap.Edit != "enter" {
		t.Errorf("other keys should stay default: %q", s.Keymap.Edit)
	}
	if s.List.Brand.Color != "#ff0000" {
		t.Errorf("brand color override lost: %q", s.List.Brand.Color)
	}
	if s.List.Help.Color != "#565f89" {
		t.Errorf("other colors should stay default: %q", s.List.Help.Color)
	}
}

func TestLoadSettingsInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[[[ bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadSettings(Paths{Root: dir})
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestLoadSettingsBadInterval(t *testing.T) {
	dir := t.TempDir()
	body := `[list]
refresh_interval = "10s"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadSettings(Paths{Root: dir})
	if err == nil {
		t.Error("expected type error for non-integer interval")
	}
}

func TestLoadSettingsZeroIntervalFallsBack(t *testing.T) {
	dir := t.TempDir()
	body := `[list]
refresh_interval = 0
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSettings(Paths{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if s.List.RefreshInterval != DefaultSettings().List.RefreshInterval {
		t.Errorf("zero interval should fall back to default, got %d", s.List.RefreshInterval)
	}
}

func TestLoadSettingsEmptyKeyFallsBack(t *testing.T) {
	dir := t.TempDir()
	body := `[keymap]
new = ""
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSettings(Paths{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if s.Keymap.New != "n" {
		t.Errorf("empty override should fall back to default, got %q", s.Keymap.New)
	}
}
