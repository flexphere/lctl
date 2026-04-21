package editor

import (
	"strings"
	"testing"
)

func TestResolveVisualWins(t *testing.T) {
	t.Setenv("VISUAL", "nvim")
	t.Setenv("EDITOR", "vim")
	got := Resolve()
	if got[0] != "nvim" {
		t.Errorf("VISUAL should win, got %v", got)
	}
}

func TestResolveFallsBackToEditor(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "nano")
	if Resolve()[0] != "nano" {
		t.Error("expected nano")
	}
}

func TestResolveFallsBackToVi(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	if Resolve()[0] != "vi" {
		t.Error("expected vi fallback")
	}
}

func TestResolveHandlesFlags(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "code -w")
	got := Resolve()
	if len(got) != 2 || got[0] != "code" || got[1] != "-w" {
		t.Errorf("unexpected split: %v", got)
	}
}

func TestCommandBuilds(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vi")
	cmd, err := Command("/tmp/x.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(cmd.Args[len(cmd.Args)-1], "x.yaml") {
		t.Errorf("file not last arg: %v", cmd.Args)
	}
}

func TestCommandRejectsEmptyPath(t *testing.T) {
	if _, err := Command(""); err == nil {
		t.Error("expected error for empty path")
	}
}
