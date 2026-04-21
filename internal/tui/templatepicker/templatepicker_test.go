package templatepicker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/lctl/internal/config"
	"github.com/flexphere/lctl/internal/tui/common"
)

func seedDir(t *testing.T, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("label: x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestListReadsYamlFiles(t *testing.T) {
	dir := seedDir(t, "a.yaml", "b.yml", "c.txt")
	got := list(dir)
	if len(got) != 2 {
		t.Fatalf("want 2, got %d", len(got))
	}
	if got[0].Name != "a.yaml" || got[1].Name != "b.yml" {
		t.Errorf("order/name wrong: %+v", got)
	}
}

func TestListMissingDirReturnsEmpty(t *testing.T) {
	if got := list("/definitely/not/a/dir"); got != nil {
		t.Errorf("want nil, got %v", got)
	}
}

func TestEnterEmitsChosen(t *testing.T) {
	dir := seedDir(t, "one.yaml", "two.yaml")
	m := New(dir, config.TemplatesSettings{})
	// Simulate Init-derived load.
	m, _ = m.Update(loadedMsg{templates: list(dir)})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg, ok := cmd().(TemplateChosenMsg)
	if !ok {
		t.Fatalf("unexpected type %T", cmd())
	}
	if msg.Template.Name != "two.yaml" {
		t.Errorf("expected two.yaml, got %q", msg.Template.Name)
	}
}

func TestEscReturnsToDashboard(t *testing.T) {
	m := New(t.TempDir(), config.TemplatesSettings{})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	sw, ok := cmd().(common.SwitchScreenMsg)
	if !ok || sw.Target != common.ScreenDashboard {
		t.Errorf("expected switch to Dashboard, got %+v", sw)
	}
}

func TestViewEmpty(t *testing.T) {
	m := New(t.TempDir(), config.TemplatesSettings{})
	m, _ = m.Update(loadedMsg{})
	if !strings.Contains(m.View(), "no *.yaml") {
		t.Errorf("expected empty hint:\n%s", m.View())
	}
}

func TestViewShowsTemplates(t *testing.T) {
	dir := seedDir(t, "alpha.yaml", "beta.yaml")
	m := New(dir, config.TemplatesSettings{})
	m, _ = m.Update(loadedMsg{templates: list(dir)})
	out := m.View()
	if !strings.Contains(out, "alpha.yaml") || !strings.Contains(out, "beta.yaml") {
		t.Errorf("templates missing:\n%s", out)
	}
}

func TestCustomTextAndColorApplied(t *testing.T) {
	dir := seedDir(t, "alpha.yaml")
	m := New(dir, config.TemplatesSettings{
		Selected:   config.TemplateRowSettings{Text: "▶ ", Color: "#bb9af7"},
		Unselected: config.TemplateRowSettings{Text: "· ", Color: "#c0caf5"},
	})
	m, _ = m.Update(loadedMsg{templates: list(dir)})
	out := m.View()
	if !strings.Contains(out, "▶ alpha.yaml") {
		t.Errorf("selected prefix not rendered:\n%s", out)
	}
	if m.styles.selectedText != "▶ " {
		t.Errorf("selected text not stored: %q", m.styles.selectedText)
	}
	if m.styles.unselectedText != "· " {
		t.Errorf("unselected text not stored: %q", m.styles.unselectedText)
	}
	if got := m.styles.selectedStyle.GetForeground(); got == nil {
		t.Error("selected style should carry a foreground")
	}
}

func TestCursorBoundaries(t *testing.T) {
	dir := seedDir(t, "a.yaml", "b.yaml")
	m := New(dir, config.TemplatesSettings{})
	m, _ = m.Update(loadedMsg{templates: list(dir)})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("Up at top should stay: %d", m.cursor)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != len(m.templates)-1 {
		t.Errorf("Down at bottom should stay: %d", m.cursor)
	}
}
