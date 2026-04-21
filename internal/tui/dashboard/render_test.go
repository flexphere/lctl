package dashboard

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/lctl/internal/launchd"
	"github.com/flexphere/lctl/internal/plist"
	"github.com/flexphere/lctl/internal/service"
)

type stubLoader struct {
	res service.ListResult
	err error
}

func (s stubLoader) List(context.Context) (service.ListResult, error) { return s.res, s.err }

// TestViewRendersAllRows exercises the Dashboard end-to-end without a
// terminal: feed WindowSize + JobsLoaded messages and assert each job
// label appears in the rendered string.
func TestViewRendersAllRows(t *testing.T) {
	jobs := []service.Job{
		{Label: "com.flexphere.one", State: launchd.StateRunning, Kind: plist.ScheduleOnLoad},
		{Label: "com.flexphere.two", State: launchd.StateLoaded, Kind: plist.ScheduleInterval},
		{Label: "com.flexphere.three", State: launchd.StateErrored, Kind: plist.SchedulePeriodic},
	}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, nil, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})
	out := m.View()
	for _, j := range jobs {
		if !strings.Contains(out, j.Label) {
			t.Errorf("label %q missing from view:\n%s", j.Label, out)
		}
	}
	if !strings.Contains(out, "3 agents") {
		t.Errorf("status missing from view:\n%s", out)
	}
}

func TestViewShowsLoadError(t *testing.T) {
	m := New(nil, nil, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{Err: errTest})
	out := m.View()
	if !strings.Contains(out, "load error") {
		t.Errorf("expected load error banner:\n%s", out)
	}
}

func TestViewShowsEmpty(t *testing.T) {
	m := New(stubLoader{}, nil, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{})
	out := m.View()
	if !strings.Contains(out, "no agents") {
		t.Errorf("expected empty state banner:\n%s", out)
	}
}

func TestViewShowsPlistIssues(t *testing.T) {
	m := New(nil, nil, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{
		Jobs:        []service.Job{{Label: "com.flexphere.ok", State: launchd.StateLoaded}},
		PlistIssues: []error{errTest},
	}})
	out := m.View()
	if !strings.Contains(out, "plist issue") {
		t.Errorf("expected plist issue banner:\n%s", out)
	}
}

type testErr string

func (e testErr) Error() string { return string(e) }

const errTest = testErr("boom")
