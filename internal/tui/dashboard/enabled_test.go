package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/lctl/internal/config"
	"github.com/flexphere/lctl/internal/launchd"
	"github.com/flexphere/lctl/internal/plist"
	"github.com/flexphere/lctl/internal/service"
)

func TestEnabledLabel(t *testing.T) {
	if enabledLabel(false) != "enabled" {
		t.Errorf("false should render as 'enabled', got %q", enabledLabel(false))
	}
	if enabledLabel(true) != "disabled" {
		t.Errorf("true should render as 'disabled', got %q", enabledLabel(true))
	}
}

func TestToItemsPropagatesDisabled(t *testing.T) {
	items := toItems([]service.Job{
		{Label: "lctl.com.on", State: launchd.StateLoaded, Disabled: false},
		{Label: "lctl.com.off", State: launchd.StateLoaded, Disabled: true},
	})
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	if items[0].(jobItem).disabled {
		t.Error("first job should be enabled")
	}
	if !items[1].(jobItem).disabled {
		t.Error("second job should be disabled")
	}
}

func TestFilterValueIncludesEnabledToken(t *testing.T) {
	fv := jobItem{display: "x", state: launchd.StateLoaded, schedule: "-", disabled: true}.FilterValue()
	if !strings.Contains(fv, "disabled") {
		t.Errorf("FilterValue should include 'disabled' when job is disabled: %q", fv)
	}
	fv = jobItem{display: "x", state: launchd.StateLoaded, schedule: "-", disabled: false}.FilterValue()
	if !strings.Contains(fv, "enabled") {
		t.Errorf("FilterValue should include 'enabled' when job is enabled: %q", fv)
	}
}

func TestHeaderLineContainsAllColumnTitles(t *testing.T) {
	defaults := config.DefaultSettings()
	r := newRowRenderer(defaults.List.Layout, defaults.List.Variables)
	h := r.renderHeader()
	for _, col := range []string{"LABEL", "STATE", "EXIT", "NEXT RUN", "SCHEDULE", "ENABLED"} {
		if !strings.Contains(h, col) {
			t.Errorf("header missing %q:\n%s", col, h)
		}
	}
}

func TestViewRendersHeaderAndEnabledColumn(t *testing.T) {
	jobs := []service.Job{
		{Label: "lctl.com.on", State: launchd.StateLoaded, Kind: plist.ScheduleOnLoad, Disabled: false},
		{Label: "lctl.com.off", State: launchd.StateLoaded, Kind: plist.ScheduleOnLoad, Disabled: true},
	}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, nil, nil, Settings{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 180, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})
	out := m.View()
	if !strings.Contains(out, "LABEL") || !strings.Contains(out, "ENABLED") {
		t.Errorf("view missing header titles:\n%s", out)
	}
	if !strings.Contains(out, "on") {
		t.Errorf("view missing enabled (on) cell:\n%s", out)
	}
	if !strings.Contains(out, "off") {
		t.Errorf("view missing disabled (off) cell:\n%s", out)
	}
}
