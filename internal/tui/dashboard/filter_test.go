package dashboard

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/lctl/internal/launchd"
	"github.com/flexphere/lctl/internal/plist"
	"github.com/flexphere/lctl/internal/service"
)

func filledModel(t *testing.T, jobs []service.Job) Model {
	t.Helper()
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, nil, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})
	return m
}

// TestSlashStartsFilter checks that pressing `/` hands control to the
// list's filter prompt so our shortcut keys don't intercept the
// incremental search input.
func TestSlashStartsFilter(t *testing.T) {
	m := filledModel(t, []service.Job{
		{Label: "lctl.com.flexphere.one"},
	})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if m.list.FilterState() != list.Filtering {
		t.Errorf("expected filter state to be Filtering, got %v", m.list.FilterState())
	}
}

// TestShortcutInertWhileFiltering confirms that while the filter
// prompt is active, our per-row shortcuts don't fire — the characters
// belong to the search term instead.
func TestShortcutInertWhileFiltering(t *testing.T) {
	m := filledModel(t, []service.Job{
		{Label: "lctl.com.flexphere.only", State: launchd.StateLoaded, Kind: plist.ScheduleOnLoad},
	})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if m.list.FilterState() != list.Filtering {
		t.Fatal("setup: filter should be active")
	}
	before := m.opStatus
	// Pressing `s` while filtering should NOT trigger start.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.opStatus != before {
		t.Errorf("opStatus should not change during filter input: %q → %q", before, m.opStatus)
	}
}

// TestFilterTyping drives the built-in filter prompt with characters
// and confirms bubbles/list picks them up. We cannot assert the
// rendered row list here because bubbles/list only filters visible
// items after the filter is committed; the reliable signal is the
// filter value it captured.
func TestFilterTyping(t *testing.T) {
	jobs := []service.Job{
		{Label: "lctl.com.flexphere.alpha"},
		{Label: "lctl.com.flexphere.beta"},
		{Label: "lctl.com.flexphere.gamma"},
	}
	m := filledModel(t, jobs)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "bet" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if m.list.FilterValue() != "bet" {
		t.Errorf("expected filter value 'bet', got %q", m.list.FilterValue())
	}
	if m.list.FilterState() != list.Filtering {
		t.Errorf("expected Filtering state, got %v", m.list.FilterState())
	}
}

// TestFilterPromptRendersAboveHeader guards against the list's
// built-in filter bar being drawn inside the items area; the filter
// prompt must live in the status line above the column header.
func TestFilterPromptRendersAboveHeader(t *testing.T) {
	jobs := []service.Job{
		{Label: "lctl.com.flexphere.alpha"},
		{Label: "lctl.com.flexphere.beta"},
	}
	m := filledModel(t, jobs)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "be" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	out := m.View()
	headerIdx := strings.Index(out, "LABEL")
	// The filter prompt is now the itemStatusLine; look for the
	// user-typed characters as the marker.
	promptIdx := strings.Index(out, "be")
	if headerIdx < 0 {
		t.Fatalf("header missing from view:\n%s", out)
	}
	if promptIdx < 0 {
		t.Fatalf("filter prompt missing from view:\n%s", out)
	}
	if promptIdx > headerIdx {
		t.Errorf("filter prompt should appear before header, got prompt@%d header@%d:\n%s", promptIdx, headerIdx, out)
	}
}
