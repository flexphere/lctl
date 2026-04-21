// Package config resolves lctl's on-disk paths following XDG Base
// Directory conventions and prepares the directories on first use.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths groups the directories lctl writes to or reads from. Values
// are absolute.
type Paths struct {
	Root      string // $XDG_CONFIG_HOME/lctl
	Templates string // Root/templates
	Scripts   string // Root/scripts
}

// Resolve returns the user's Paths, honoring $XDG_CONFIG_HOME and
// falling back to ~/.config when unset.
func Resolve() (Paths, error) {
	base, err := xdgConfigHome()
	if err != nil {
		return Paths{}, err
	}
	root := filepath.Join(base, "lctl")
	return Paths{
		Root:      root,
		Templates: filepath.Join(root, "templates"),
		Scripts:   filepath.Join(root, "scripts"),
	}, nil
}

// Ensure creates the Root/templates/scripts directories if absent and
// seeds the default template files and an example config.toml the
// first time any of them is missing. Existing files are never
// overwritten.
func Ensure(p Paths) error {
	for _, d := range []string{p.Root, p.Templates, p.Scripts} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", d, err)
		}
	}
	if err := seedTemplates(p.Templates); err != nil {
		return err
	}
	return seedConfig(p.Root)
}

// seedConfig writes a heavily-commented config.toml at the root once
// so users can discover available options without reading the code.
func seedConfig(root string) error {
	path := filepath.Join(root, "config.toml")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte(defaultConfigTOML), 0o644)
}

// xdgConfigHome reads $XDG_CONFIG_HOME or falls back to $HOME/.config.
func xdgConfigHome() (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home: %w", err)
	}
	return filepath.Join(home, ".config"), nil
}

// seedTemplates writes the default template files for any that do not
// yet exist. Writing is idempotent and skip-if-present.
func seedTemplates(dir string) error {
	for name, content := range defaultTemplates {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("seed %s: %w", name, err)
		}
	}
	return nil
}
