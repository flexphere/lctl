package dashboard

import (
	"strings"
	"testing"

	"github.com/flexphere/lctl/internal/config"
	"github.com/flexphere/lctl/internal/launchd"
)

func TestValueTextOverridesParent(t *testing.T) {
	vars := defaultVars()
	vars["state"] = config.VariableSettings{
		Text:  "$value",
		Width: 20,
		Values: map[string]config.ValueSettings{
			"running": {Text: "● $value"},
		},
	}
	r := newRowRenderer(config.DefaultSettings().List.Layout, vars)
	out := r.renderItem(jobItem{display: "com.x", state: launchd.StateRunning}, false)
	if !strings.Contains(out, "● running") {
		t.Errorf("per-value text template not applied for running:\n%s", out)
	}
}

func TestValueTextLeavesOtherStatesToParent(t *testing.T) {
	vars := defaultVars()
	vars["state"] = config.VariableSettings{
		Text:  "[$value]",
		Width: 20,
		Values: map[string]config.ValueSettings{
			"running": {Text: "●"},
		},
	}
	r := newRowRenderer(config.DefaultSettings().List.Layout, vars)
	runningOut := r.renderItem(jobItem{display: "com.x", state: launchd.StateRunning}, false)
	erroredOut := r.renderItem(jobItem{display: "com.y", state: launchd.StateErrored}, false)
	if !strings.Contains(runningOut, "●") {
		t.Errorf("per-value template missing for running: %s", runningOut)
	}
	if !strings.Contains(erroredOut, "[errored]") {
		t.Errorf("parent template should still apply to errored: %s", erroredOut)
	}
}

func TestValueColorTakesPriority(t *testing.T) {
	vars := defaultVars()
	vars["state"] = config.VariableSettings{
		Color: "220",
		Width: 10,
		Values: map[string]config.ValueSettings{
			"errored": {Color: "196"},
		},
	}
	r := newRowRenderer(config.DefaultSettings().List.Layout, vars)
	if got := r.valueColor("state", "errored"); got != "196" {
		t.Errorf("value-level color should win: got %q want 196", got)
	}
	if got := r.valueColor("state", "running"); got != "220" {
		t.Errorf("parent color should apply when value unset: got %q want 220", got)
	}
}

func TestValueColorFallsThroughWhenBothEmpty(t *testing.T) {
	vars := defaultVars()
	vars["state"] = config.VariableSettings{
		Width: 10,
		Values: map[string]config.ValueSettings{
			"running": {Text: "●"}, // text set but color empty
		},
	}
	r := newRowRenderer(config.DefaultSettings().List.Layout, vars)
	if got := r.valueColor("state", "running"); got != "" {
		t.Errorf("empty value + empty parent → empty color; got %q", got)
	}
}

func TestCursorValueCustomization(t *testing.T) {
	vars := defaultVars()
	vars["cursor"] = config.VariableSettings{
		Text:  "$value",
		Width: 2,
		Values: map[string]config.ValueSettings{
			"selected":   {Text: "▶ ", Color: "#ff9e64"},
			"unselected": {Text: "  "},
		},
	}
	r := newRowRenderer(config.DefaultSettings().List.Layout, vars)
	selected := r.renderItem(jobItem{display: "com.x", state: launchd.StateLoaded}, true)
	unselected := r.renderItem(jobItem{display: "com.y", state: launchd.StateLoaded}, false)
	if !strings.Contains(selected, "▶") {
		t.Errorf("selected cursor should use ▶ glyph:\n%s", selected)
	}
	if strings.Contains(unselected, "▶") {
		t.Errorf("unselected cursor must not show ▶ glyph:\n%s", unselected)
	}
}

func TestCursorDefaultsRenderArrow(t *testing.T) {
	r := newRowRenderer(config.DefaultSettings().List.Layout, defaultVars())
	selected := r.renderItem(jobItem{display: "com.x", state: launchd.StateLoaded}, true)
	if !strings.Contains(selected, "▶") {
		t.Errorf("default selected cursor glyph should be ▶:\n%s", selected)
	}
}

func TestEnabledValueCustomization(t *testing.T) {
	vars := defaultVars()
	vars["enabled"] = config.VariableSettings{
		Width: 10,
		Values: map[string]config.ValueSettings{
			"enabled":  {Text: "on", Color: "42"},
			"disabled": {Text: "off", Color: "196"},
		},
	}
	r := newRowRenderer(config.DefaultSettings().List.Layout, vars)
	on := r.renderItem(jobItem{display: "com.x", state: launchd.StateRunning, disabled: false}, false)
	off := r.renderItem(jobItem{display: "com.y", state: launchd.StateRunning, disabled: true}, false)
	if !strings.Contains(on, "on") {
		t.Errorf("'enabled' should render as 'on': %s", on)
	}
	if !strings.Contains(off, "off") {
		t.Errorf("'disabled' should render as 'off': %s", off)
	}
}
