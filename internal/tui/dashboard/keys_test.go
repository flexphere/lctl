package dashboard

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/lctl/internal/launchd"
	"github.com/flexphere/lctl/internal/plist"
	"github.com/flexphere/lctl/internal/service"
)

type recordingOps struct {
	calls []string
	err   error
}

func (r *recordingOps) Start(_ context.Context, l string) error {
	r.calls = append(r.calls, "start "+l)
	return r.err
}
func (r *recordingOps) Stop(_ context.Context, l string) error {
	r.calls = append(r.calls, "stop "+l)
	return r.err
}
func (r *recordingOps) Restart(_ context.Context, l string) error {
	r.calls = append(r.calls, "restart "+l)
	return r.err
}
func (r *recordingOps) Enable(_ context.Context, l string) error {
	r.calls = append(r.calls, "enable "+l)
	return r.err
}
func (r *recordingOps) Disable(_ context.Context, l string) error {
	r.calls = append(r.calls, "disable "+l)
	return r.err
}
func (r *recordingOps) Delete(_ context.Context, l string) error {
	r.calls = append(r.calls, "delete "+l)
	return r.err
}

func loadedModel(t *testing.T, ops service.Ops) Model {
	t.Helper()
	jobs := []service.Job{
		{Label: "lctl.com.flexphere.first", State: launchd.StateLoaded, Kind: plist.ScheduleOnLoad},
		{Label: "lctl.com.flexphere.second", State: launchd.StateRunning, Kind: plist.ScheduleInterval},
	}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, ops, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})
	return m
}

// runKey delivers a KeyMsg to the model, invokes any returned command
// to produce a follow-up message, and dispatches that message too.
// This matches Bubble Tea's runtime well enough for unit testing.
func runKey(t *testing.T, m Model, key string) Model {
	t.Helper()
	var msg tea.Msg
	switch key {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	m, cmd := m.Update(msg)
	if cmd == nil {
		return m
	}
	out := cmd()
	if out == nil {
		return m
	}
	m, _ = m.Update(out)
	return m
}

func TestKeyStartInvokesOps(t *testing.T) {
	ops := &recordingOps{}
	m := loadedModel(t, ops)
	runKey(t, m, "s")
	if len(ops.calls) == 0 || ops.calls[0] != "start lctl.com.flexphere.first" {
		t.Errorf("expected prefixed start call, got %v", ops.calls)
	}
}

func TestKeyStopInvokesOps(t *testing.T) {
	ops := &recordingOps{}
	m := loadedModel(t, ops)
	runKey(t, m, "x")
	if len(ops.calls) == 0 || !strings.HasPrefix(ops.calls[0], "stop ") {
		t.Errorf("expected stop call: %v", ops.calls)
	}
}

func TestKeyRestartInvokesOps(t *testing.T) {
	ops := &recordingOps{}
	m := loadedModel(t, ops)
	runKey(t, m, "r")
	if len(ops.calls) == 0 || !strings.HasPrefix(ops.calls[0], "restart ") {
		t.Errorf("expected restart call: %v", ops.calls)
	}
}

func TestKeyToggleFromEnabledInvokesDisable(t *testing.T) {
	ops := &recordingOps{}
	jobs := []service.Job{
		{Label: "lctl.com.flexphere.first", State: launchd.StateLoaded, Kind: plist.ScheduleOnLoad, Disabled: false},
	}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, ops, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	_ = m2
	if cmd == nil {
		t.Fatal("expected command from toggle")
	}
	out := cmd()
	if _, ok := out.(OpCompletedMsg); !ok {
		t.Fatalf("expected OpCompletedMsg, got %T", out)
	}
	if len(ops.calls) != 1 || !strings.HasPrefix(ops.calls[0], "disable ") {
		t.Errorf("expected disable call (enabled → disable), got %v", ops.calls)
	}
}

func TestKeyToggleFromDisabledInvokesEnable(t *testing.T) {
	ops := &recordingOps{}
	jobs := []service.Job{
		{Label: "lctl.com.flexphere.first", State: launchd.StateLoaded, Kind: plist.ScheduleOnLoad, Disabled: true},
	}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, ops, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	_ = m2
	if cmd == nil {
		t.Fatal("expected command from toggle")
	}
	_ = cmd()
	if len(ops.calls) != 1 || !strings.HasPrefix(ops.calls[0], "enable ") {
		t.Errorf("expected enable call (disabled → enable), got %v", ops.calls)
	}
}

func TestKeyEnterWithoutFlowShowsHint(t *testing.T) {
	m := loadedModel(t, &recordingOps{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(m.opStatus, "flow not configured") {
		t.Errorf("expected flow-missing hint, got %q", m.opStatus)
	}
}

func TestKeyOpsNilDegrade(t *testing.T) {
	m := loadedModel(t, nil)
	m = runKey(t, m, "s")
	if !strings.Contains(m.opStatus, "not configured") {
		t.Errorf("expected nil-ops status hint, got %q", m.opStatus)
	}
}

func TestKeyOpErrorReflectedInStatus(t *testing.T) {
	ops := &recordingOps{err: errors.New("boom")}
	m := loadedModel(t, ops)
	m = runKey(t, m, "s")
	if !strings.Contains(m.opStatus, "boom") {
		t.Errorf("expected error in status: %q", m.opStatus)
	}
}

func TestKeyRefresh(t *testing.T) {
	m := loadedModel(t, nil)
	m.opStatus = "stale"
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	m, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("expected command to reload")
	}
	if m.opStatus != "" {
		t.Errorf("opStatus should clear, got %q", m.opStatus)
	}
}

func TestEnterWithNoSelectionDoesNothing(t *testing.T) {
	m := New(stubLoader{}, nil, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Errorf("expected no command when no row selected")
	}
}
