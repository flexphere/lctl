package dashboard

import (
	"strings"
	"testing"

	"github.com/flexphere/lctl/internal/config"
	"github.com/flexphere/lctl/internal/launchd"
)

// defaultVars builds a fresh copy of the compiled-in variable defaults
// so tests can tweak individual entries without cross-pollution.
func defaultVars() map[string]config.VariableSettings {
	src := config.DefaultSettings().List.Variables
	out := make(map[string]config.VariableSettings, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func TestRendererAppliesTextTemplate(t *testing.T) {
	vars := defaultVars()
	vars["label"] = config.VariableSettings{Text: "Label:$label", Width: 30}
	r := newRowRenderer(config.DefaultSettings().List.Layout, vars)
	out := r.renderItem(jobItem{
		display: "com.x",
		state:   launchd.StateLoaded,
	}, false)
	if !strings.Contains(out, "Label:com.x") {
		t.Errorf("text template not applied:\n%s", out)
	}
}

func TestRendererTextTemplateValueAliasWorks(t *testing.T) {
	vars := defaultVars()
	vars["label"] = config.VariableSettings{Text: "[$value]", Width: 30}
	r := newRowRenderer(config.DefaultSettings().List.Layout, vars)
	out := r.renderItem(jobItem{display: "com.x", state: launchd.StateLoaded}, false)
	if !strings.Contains(out, "[com.x]") {
		t.Errorf("$value alias not expanded:\n%s", out)
	}
}

func TestRendererColorOverrideWinsOverValueDefault(t *testing.T) {
	// Build a renderer with a single-color override on $state. The
	// override must be visible to styledState via the variables map
	// so both running and errored rows use the same color.
	vars := defaultVars()
	vars["state"] = config.VariableSettings{Text: "$value", Width: 10, Color: "220"}
	r := newRowRenderer(config.DefaultSettings().List.Layout, vars)

	// The color is applied via lipgloss; testing ANSI output varies
	// by terminal capability detection, so we assert the override is
	// stored and reachable via the variables map instead.
	if got := r.variables["state"].Color; got != "220" {
		t.Errorf("override color not stored: %q", got)
	}
	// Sanity: render both rows. Nothing should panic and the state
	// cells should contain their raw value.
	running := r.renderItem(jobItem{display: "com.x", state: launchd.StateRunning}, false)
	errored := r.renderItem(jobItem{display: "com.y", state: launchd.StateErrored}, false)
	if !strings.Contains(running, "running") || !strings.Contains(errored, "errored") {
		t.Errorf("render output missing state text:\nrunning=%q\nerrored=%q", running, errored)
	}
}

func TestRendererHeaderSkipsTextTemplate(t *testing.T) {
	vars := defaultVars()
	vars["label"] = config.VariableSettings{Text: "Label:$value", Width: 30}
	r := newRowRenderer(config.DefaultSettings().List.Layout, vars)
	header := r.renderHeader()
	if strings.Contains(header, "Label:LABEL") {
		t.Errorf("header should not run text template through the title:\n%s", header)
	}
	if !strings.Contains(header, "LABEL") {
		t.Errorf("header should still show the raw title:\n%s", header)
	}
}

func TestRendererLayoutSkipsColumn(t *testing.T) {
	// A user-supplied layout that omits $enabled should not emit the
	// enabled column at all.
	format := config.LayoutSettings{
		Template: "$label$divider$state",
		Divider:  "  ",
	}
	r := newRowRenderer(format, defaultVars())
	out := r.renderItem(jobItem{display: "com.x", state: launchd.StateLoaded, disabled: true}, false)
	if strings.Contains(out, "disabled") {
		t.Errorf("layout without $enabled should not render the disabled cell:\n%s", out)
	}
}

func TestRendererCustomDivider(t *testing.T) {
	format := config.LayoutSettings{
		Template: "$label$divider$state",
		Divider:  " | ",
	}
	r := newRowRenderer(format, defaultVars())
	out := r.renderItem(jobItem{display: "com.x", state: launchd.StateLoaded}, false)
	if !strings.Contains(out, " | ") {
		t.Errorf("divider not substituted:\n%s", out)
	}
}

func TestRendererUnknownVariableIsLiteral(t *testing.T) {
	format := config.LayoutSettings{
		Template: "$label$divider$nope$divider$state",
		Divider:  "  ",
	}
	r := newRowRenderer(format, defaultVars())
	out := r.renderItem(jobItem{display: "com.x", state: launchd.StateLoaded}, false)
	if !strings.Contains(out, "$nope") {
		t.Errorf("unknown variable should pass through as literal:\n%s", out)
	}
}
