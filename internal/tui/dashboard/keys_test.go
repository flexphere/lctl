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

func TestDeleteArmsConfirmBeforeOp(t *testing.T) {
	ops := &recordingOps{}
	m := loadedModel(t, ops)
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	if cmd != nil {
		t.Errorf("arming delete should not dispatch a command, got %T", cmd())
	}
	if m.pendingDelete == "" {
		t.Error("expected pendingDelete to be set after D")
	}
	if len(ops.calls) != 0 {
		t.Errorf("delete must not fire before confirm, got %v", ops.calls)
	}
	if !strings.Contains(m.View(), "delete") || !strings.Contains(m.View(), "(y/N)") {
		t.Errorf("view should prompt for y/N:\n%s", m.View())
	}
}

func TestDeleteConfirmYRunsOp(t *testing.T) {
	ops := &recordingOps{}
	m := loadedModel(t, ops)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected delete command after y")
	}
	_ = cmd()
	if m.pendingDelete != "" {
		t.Errorf("pendingDelete should clear on confirm, got %q", m.pendingDelete)
	}
	if len(ops.calls) != 1 || !strings.HasPrefix(ops.calls[0], "delete ") {
		t.Errorf("expected delete call, got %v", ops.calls)
	}
}

func TestDeleteConfirmCancels(t *testing.T) {
	ops := &recordingOps{}
	m := loadedModel(t, ops)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	// Any non-affirmative key cancels. Use "n" as the canonical choice.
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd != nil {
		t.Errorf("cancel should not dispatch a command, got %T", cmd())
	}
	if m.pendingDelete != "" {
		t.Errorf("pendingDelete should clear on cancel, got %q", m.pendingDelete)
	}
	if len(ops.calls) != 0 {
		t.Errorf("cancel must not fire delete, got %v", ops.calls)
	}
	if !strings.Contains(m.opStatus, "cancelled") {
		t.Errorf("expected 'cancelled' in status, got %q", m.opStatus)
	}
}

func TestDeleteConfirmBlocksOtherOps(t *testing.T) {
	ops := &recordingOps{}
	m := loadedModel(t, ops)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	// While prompt is armed, "s" should cancel the prompt without
	// starting the job — the start path is gated behind confirmation
	// resolution.
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd != nil {
		_ = cmd()
	}
	if len(ops.calls) != 0 {
		t.Errorf("start must not fire while delete prompt is armed, got %v", ops.calls)
	}
	if m.pendingDelete != "" {
		t.Errorf("pendingDelete should clear after any key, got %q", m.pendingDelete)
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
