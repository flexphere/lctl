package dashboard

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/lctl/internal/config"
	"github.com/flexphere/lctl/internal/launchd"
	"github.com/flexphere/lctl/internal/plist"
	"github.com/flexphere/lctl/internal/service"
)

func setupLoaded(t *testing.T, settings Settings, ops service.Ops) Model {
	t.Helper()
	jobs := []service.Job{
		{Label: "lctl.com.flexphere.only", State: launchd.StateRunning, Kind: plist.ScheduleOnLoad},
	}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, ops, nil, settings)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})
	return m
}

func TestCustomKeymapTriggersOps(t *testing.T) {
	settings := Settings{
		Keymap: config.Keymap{
			New: "a", Edit: "b", Log: "c",
			Start: "S", Stop: "T", Restart: "U",
			Toggle: "E", Delete: "X",
			Refresh: "Y", Quit: "Z",
		},
	}
	ops := &recordingOps{}
	m := setupLoaded(t, settings, ops)
	// Custom 'S' → start
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}}
	m2, cmd := m.Update(msg)
	_ = m2
	if cmd == nil {
		t.Fatal("expected command from custom start key")
	}
	out := cmd()
	if _, ok := out.(OpCompletedMsg); !ok {
		t.Fatalf("want OpCompletedMsg, got %T", out)
	}
	if len(ops.calls) != 1 || ops.calls[0] != "start lctl.com.flexphere.only" {
		t.Errorf("start not invoked with prefix: %v", ops.calls)
	}
}

func TestDefaultKeyNoLongerWorksWhenRemapped(t *testing.T) {
	settings := Settings{
		Keymap: config.Keymap{
			New: "a", Edit: "b", Log: "c",
			Start: "S", Stop: "T", Restart: "U",
			Toggle: "E", Delete: "X",
			Refresh: "Y", Quit: "Z",
		},
	}
	ops := &recordingOps{}
	m := setupLoaded(t, settings, ops)
	// Pressing lowercase 's' (old default) should not start anything.
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	_, cmd := m.Update(msg)
	if cmd != nil {
		out := cmd()
		if _, ok := out.(OpCompletedMsg); ok {
			t.Errorf("old default key should be inert after remap")
		}
	}
}

func TestCtrlCQuitsEvenWhenRemapped(t *testing.T) {
	settings := Settings{
		Keymap: config.Keymap{Quit: "x"},
	}
	m := setupLoaded(t, settings, &recordingOps{})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c should still quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg from ctrl+c")
	}
}

func TestAutoRefreshInitReturnsBatch(t *testing.T) {
	settings := Settings{
		AutoRefresh:     true,
		RefreshInterval: 50 * time.Millisecond,
		Keymap:          config.DefaultSettings().Keymap,
	}
	m := New(stubLoader{}, nil, nil, settings)
	if m.Init() == nil {
		t.Fatal("Init should return a command when auto-refresh is on")
	}
}

func TestAutoRefreshMsgSchedulesNextTick(t *testing.T) {
	settings := Settings{
		AutoRefresh:     true,
		RefreshInterval: 10 * time.Millisecond,
	}
	m := New(stubLoader{}, nil, nil, settings)
	_, cmd := m.Update(autoRefreshMsg{})
	if cmd == nil {
		t.Fatal("expected chain of load + tick")
	}
}

func TestAutoRefreshMsgIgnoredWhenDisabled(t *testing.T) {
	m := New(stubLoader{}, nil, nil, Settings{})
	_, cmd := m.Update(autoRefreshMsg{})
	if cmd != nil {
		t.Error("tick must be no-op when auto-refresh disabled")
	}
}

func TestSettingsZeroFallsBackToDefaults(t *testing.T) {
	m := New(stubLoader{}, nil, nil, Settings{})
	if m.keys.Edit == "" || m.interval <= 0 {
		t.Errorf("zero Settings should use defaults, got %+v interval=%v", m.keys, m.interval)
	}
}

func TestHelpLineMentionsCustomKey(t *testing.T) {
	settings := Settings{Keymap: config.Keymap{
		New: "N", Edit: "E", Log: "L", Start: "s", Stop: "x", Restart: "r",
		Toggle: "tab", Delete: "D",
		Refresh: "ctrl+r", Quit: "q",
	}}
	m := New(stubLoader{}, nil, nil, settings)
	if want := "N:new"; !contains(m.helpLine(), want) {
		t.Errorf("help line missing custom key %q: %s", want, m.helpLine())
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
