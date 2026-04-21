package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/flexphere/lctl/internal/launchd"
	"github.com/flexphere/lctl/internal/plist"
	"github.com/flexphere/lctl/internal/service"
)

func TestAggregateTitleLayout(t *testing.T) {
	jobs := []service.Job{
		{Label: "lctl.com.flexphere.foo", State: launchd.StateLoaded, Kind: plist.ScheduleOnLoad},
	}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, nil, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})

	out := m.View()
	firstLine := out
	if idx := strings.Index(out, "\n"); idx >= 0 {
		firstLine = out[:idx]
	}
	visibleWidth := lipgloss.Width(firstLine)
	if visibleWidth != 120 {
		t.Errorf("title line should span 120 cols, got %d:\n%q", visibleWidth, firstLine)
	}

	// Brand sits on the left, aggregate lands on the right.
	stripped := ansiStrip(firstLine)
	if !strings.HasPrefix(stripped, "lctl") {
		t.Errorf("title should start with 'lctl' brand, got %q", stripped[:min(len(stripped), 8)])
	}
	if !strings.HasSuffix(strings.TrimRight(stripped, " "), "errored") {
		t.Errorf("title should end (after trimming trailing spaces) with 'errored', got %q", stripped)
	}

	// There should be a run of spaces between the brand and the
	// aggregate segment so they sit at opposite edges.
	if !strings.Contains(stripped, "lctl  ") && !strings.Contains(stripped, "lctl   ") {
		t.Errorf("brand and aggregate should be separated by padding spaces: %q", stripped)
	}
}

// ansiStrip removes CSI escape sequences from s. Good enough for
// tests that only need to inspect visible content.
func ansiStrip(s string) string {
	b := strings.Builder{}
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < '@' || s[j] > '~') {
				j++
			}
			i = j + 1
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
