package dashboard

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/flexphere/lctl/internal/launchd"
	"github.com/flexphere/lctl/internal/plist"
	"github.com/flexphere/lctl/internal/service"
)

// TestLabelStretchesToFillTerminalWidth confirms that when the
// terminal is wider than the layout's natural width, the extra space
// is added to the $label column so the rendered row reaches the edge.
func TestLabelStretchesToFillTerminalWidth(t *testing.T) {
	jobs := []service.Job{
		{Label: "lctl.com.flexphere.one", State: launchd.StateLoaded, Kind: plist.ScheduleOnLoad},
	}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, nil, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 200, Height: 30})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})

	headerLine := m.renderer.renderHeader()
	// Without stretching the header would be ~defaults (cursor 2 +
	// label 40 + ... ≈ 122). Width 200 should push it well beyond.
	if w := lipgloss.Width(headerLine); w < 180 {
		t.Errorf("expected header to stretch close to 200 cols, got %d:\n%s", w, headerLine)
	}
	if w := m.renderer.widthOf("label"); w <= defaultLabelWidth {
		t.Errorf("label width should exceed the default when stretching, got %d", w)
	}
}

// TestNoStretchingWhenTerminalNarrow guarantees the stretch path is a
// no-op on small terminals; we never shrink below the configured width.
func TestNoStretchingWhenTerminalNarrow(t *testing.T) {
	jobs := []service.Job{{Label: "lctl.com.flexphere.one"}}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, nil, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 30})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})

	if w := m.renderer.widthOf("label"); w != defaultLabelWidth {
		t.Errorf("label width should stay at the configured default when terminal is narrow, got %d", w)
	}
}

// TestStretchOnlyAffectsLabel makes sure other columns keep their
// configured widths regardless of the surplus.
func TestStretchOnlyAffectsLabel(t *testing.T) {
	jobs := []service.Job{{Label: "lctl.com.flexphere.one"}}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, nil, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 240, Height: 30})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})

	if w := m.renderer.widthOf("state"); w != defaultStateWidth {
		t.Errorf("state width should stay %d, got %d", defaultStateWidth, w)
	}
	if w := m.renderer.widthOf("next_run"); w != defaultNextRunWidth {
		t.Errorf("next_run width should stay %d, got %d", defaultNextRunWidth, w)
	}
}
