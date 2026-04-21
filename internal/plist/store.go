package plist

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Store reads and writes launchd .plist files under a base directory,
// typically ~/Library/LaunchAgents.
type Store struct {
	Dir string
}

// NewUserStore returns a Store rooted at ~/Library/LaunchAgents, creating
// the directory if it does not yet exist.
func NewUserStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home: %w", err)
	}
	dir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("ensure dir: %w", err)
	}
	return &Store{Dir: dir}, nil
}

// Path returns the absolute plist path for the given label.
func (s *Store) Path(label string) (string, error) {
	if strings.ContainsAny(label, `/\`) || strings.Contains(label, "..") {
		return "", fmt.Errorf("invalid label: %q", label)
	}
	if strings.TrimSpace(label) == "" {
		return "", errors.New("label must not be empty")
	}
	return filepath.Join(s.Dir, label+".plist"), nil
}

// List returns all agents found in the store directory. I/O or parse
// errors for individual files are collected and returned as a second
// value so the caller can surface them without losing the good entries.
func (s *Store) List() ([]*Agent, []error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, []error{fmt.Errorf("read dir: %w", err)}
	}
	var agents []*Agent
	var errs []error
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".plist") {
			continue
		}
		path := filepath.Join(s.Dir, e.Name())
		a, err := s.readFile(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", e.Name(), err))
			continue
		}
		agents = append(agents, a)
	}
	return agents, errs
}

// Load reads the plist for the given label.
func (s *Store) Load(label string) (*Agent, error) {
	p, err := s.Path(label)
	if err != nil {
		return nil, err
	}
	return s.readFile(p)
}

// Save writes the agent atomically via rename. The label must match the
// destination filename.
func (s *Store) Save(a *Agent) error {
	if a == nil {
		return errors.New("agent is nil")
	}
	p, err := s.Path(a.Label)
	if err != nil {
		return err
	}
	data, err := EncodeBytes(a)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.Dir, ".lctl-*.plist.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, p); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	cleanup = false
	return nil
}

// Delete removes the plist for the given label. Missing file is not an error.
func (s *Store) Delete(label string) error {
	p, err := s.Path(label)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("remove: %w", err)
	}
	return nil
}

func (s *Store) readFile(path string) (*Agent, error) {
	f, err := os.Open(path) //nolint:gosec // path is derived from validated label or directory listing
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer func() { _ = f.Close() }()
	return Decode(f)
}
