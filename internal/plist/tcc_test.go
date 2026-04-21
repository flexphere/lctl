package plist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTCCPrefixDoesNotFalseMatch(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	// Build an agent whose paths just look similar to TCC-sensitive
	// dirs but aren't actually under them ("Documents-backup" etc).
	bogus := filepath.Join(home, "Documents-backup", "run.sh")
	a := &Agent{
		Label:            "com.flexphere.tcc",
		Program:          "/bin/true",
		WorkingDirectory: bogus,
		RunAtLoad:        true,
	}
	r := Validate(a)
	for _, i := range r.Issues {
		if i.Warning && strings.Contains(i.Message, "Documents") {
			t.Errorf("false TCC match on %q: %+v", bogus, i)
		}
	}
}

func TestTCCMatchesExactDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	a := &Agent{
		Label:            "com.flexphere.tcc2",
		Program:          "/bin/true",
		WorkingDirectory: filepath.Join(home, "Documents"),
		RunAtLoad:        true,
	}
	r := Validate(a)
	found := false
	for _, i := range r.Issues {
		if i.Warning && strings.Contains(i.Message, "Documents") {
			found = true
		}
	}
	if !found {
		t.Error("expected TCC warning for exact Documents match")
	}
}

func TestTCCMatchesDirectoryContents(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	a := &Agent{
		Label:             "com.flexphere.tcc3",
		Program:           "/bin/true",
		StandardOutPath:   filepath.Join(home, "Documents", "logs", "out.log"),
		StandardErrorPath: filepath.Join(home, "Library", "Mail", "err.log"),
		RunAtLoad:         true,
	}
	r := Validate(a)
	var stdout, mail bool
	for _, i := range r.Issues {
		if i.Warning && strings.Contains(i.Message, "Documents") && i.Field == "StandardOutPath" {
			stdout = true
		}
		if i.Warning && strings.Contains(i.Message, "Library/Mail") && i.Field == "StandardErrorPath" {
			mail = true
		}
	}
	if !stdout || !mail {
		t.Errorf("expected TCC warnings for stdout and mail: %+v", r.Issues)
	}
}
