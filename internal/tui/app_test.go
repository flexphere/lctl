package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/lctl/internal/config"
	"github.com/flexphere/lctl/internal/service"
	"github.com/flexphere/lctl/internal/tui/common"
	"github.com/flexphere/lctl/internal/tui/dashboard"
	"github.com/flexphere/lctl/internal/tui/templatepicker"
)

type nilLoader struct{}

func (nilLoader) List(context.Context) (service.ListResult, error) {
	return service.ListResult{}, nil
}

func TestAppRoutesToDashboard(t *testing.T) {
	a := New(nilLoader{}, nil, nil, "", dashboard.Settings{}, config.TemplatesSettings{})
	if a.current != common.ScreenDashboard {
		t.Errorf("expected dashboard as initial screen, got %v", a.current)
	}
	// bubbles/list needs a non-zero size before it fully renders.
	next, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// The dashboard View drops the title but always includes the
	// column header — use it as the proof-of-render signal.
	if !strings.Contains(next.(App).View(), "LABEL") {
		t.Errorf("expected column header in view: %s", next.(App).View())
	}
}

func TestAppHandlesSwitchToTemplatePicker(t *testing.T) {
	a := New(nilLoader{}, nil, nil, "", dashboard.Settings{}, config.TemplatesSettings{})
	next, _ := a.Update(common.SwitchScreenMsg{Target: common.ScreenTemplatePicker})
	app := next.(App)
	if app.current != common.ScreenTemplatePicker {
		t.Errorf("expected screen TemplatePicker, got %v", app.current)
	}
}

func TestAppTemplateChosenReturnsToDashboard(t *testing.T) {
	a := New(nilLoader{}, nil, nil, "", dashboard.Settings{}, config.TemplatesSettings{})
	a.current = common.ScreenTemplatePicker
	next, _ := a.Update(templatepicker.TemplateChosenMsg{Template: templatepicker.Template{Name: "x.yaml", Path: "/tmp/x.yaml"}})
	app := next.(App)
	if app.current != common.ScreenDashboard {
		t.Errorf("expected return to Dashboard after pick, got %v", app.current)
	}
}

func TestAppInitReturnsCmd(t *testing.T) {
	a := New(nilLoader{}, nil, nil, "", dashboard.Settings{}, config.TemplatesSettings{})
	if cmd := a.Init(); cmd == nil {
		t.Error("Init should return initial load command")
	}
}

func TestAppUnknownScreenRendersEmpty(t *testing.T) {
	a := New(nilLoader{}, nil, nil, "", dashboard.Settings{}, config.TemplatesSettings{})
	a.current = common.Screen(999)
	if a.View() != "" {
		t.Error("unknown screen should render empty")
	}
}

func TestAppWindowSizeCached(t *testing.T) {
	a := New(nilLoader{}, nil, nil, "", dashboard.Settings{}, config.TemplatesSettings{})
	next, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := next.(App)
	if app.lastSize.Width != 120 {
		t.Errorf("expected size cached, got %+v", app.lastSize)
	}
}
