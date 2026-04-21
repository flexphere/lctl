package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/lctl/internal/launchd"
	"github.com/flexphere/lctl/internal/plist"
	"github.com/flexphere/lctl/internal/service"
)

// TestItemsStripPrefixForDisplay confirms the list never exposes the
// raw lctl. prefix to the user.
func TestItemsStripPrefixForDisplay(t *testing.T) {
	items := toItems([]service.Job{
		{Label: "lctl.com.flexphere.one", State: launchd.StateLoaded, Kind: plist.ScheduleOnLoad},
	})
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	it := items[0].(jobItem)
	if it.display != "com.flexphere.one" {
		t.Errorf("label not stripped: %q", it.display)
	}
	if it.label != "lctl.com.flexphere.one" {
		t.Errorf("full label lost: %q", it.label)
	}
}

// TestSelectedLabelAddsPrefix guarantees downstream ops get the full
// lctl-scoped label even though the table cell is stripped.
func TestSelectedLabelAddsPrefix(t *testing.T) {
	jobs := []service.Job{
		{Label: "lctl.com.flexphere.foo", State: launchd.StateLoaded, Kind: plist.ScheduleOnLoad},
	}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, nil, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})
	got := m.SelectedLabel()
	if got != "lctl.com.flexphere.foo" {
		t.Errorf("SelectedLabel must return prefixed label: %q", got)
	}
}

// TestViewRendersStrippedLabel is an end-to-end sanity check that the
// lctl. prefix does not leak to the rendered view string.
func TestViewRendersStrippedLabel(t *testing.T) {
	jobs := []service.Job{
		{Label: "lctl.com.flexphere.visible", State: launchd.StateLoaded, Kind: plist.ScheduleOnLoad},
	}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, nil, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})
	out := m.View()
	if !strings.Contains(out, "com.flexphere.visible") {
		t.Errorf("view missing stripped label:\n%s", out)
	}
	if strings.Contains(out, "lctl.com.flexphere.visible") {
		t.Errorf("view must not expose lctl. prefix:\n%s", out)
	}
}
