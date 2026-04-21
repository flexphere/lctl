package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/lctl/internal/config"
	"github.com/flexphere/lctl/internal/service"
	"github.com/flexphere/lctl/internal/tui/common"
	"github.com/flexphere/lctl/internal/tui/dashboard"
	"github.com/flexphere/lctl/internal/tui/templatepicker"
)

// App is the root Bubble Tea model. It owns the current screen and
// delegates update/view to the sub-model.
type App struct {
	current   common.Screen
	dashboard dashboard.Model
	picker    templatepicker.Model
	lastSize  tea.WindowSizeMsg
	pickerDir string
	pickerCfg config.TemplatesSettings
}

// New constructs an App rooted at the Dashboard.
func New(loader dashboard.Loader, ops service.Ops, flow *dashboard.EditFlow, templatesDir string, dash dashboard.Settings, pickerCfg config.TemplatesSettings) App {
	return App{
		current:   common.ScreenDashboard,
		dashboard: dashboard.New(loader, ops, flow, dash),
		picker:    templatepicker.New(templatesDir, pickerCfg),
		pickerDir: templatesDir,
		pickerCfg: pickerCfg,
	}
}

// Init runs initial commands.
func (a App) Init() tea.Cmd { return a.dashboard.Init() }

// Update routes messages to the active screen and handles transitions.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		a.lastSize = ws
	}
	switch m := msg.(type) {
	case common.SwitchScreenMsg:
		return a.switchScreen(m)
	case templatepicker.TemplateChosenMsg:
		a.current = common.ScreenDashboard
		var cmd tea.Cmd
		a.dashboard, cmd = a.dashboard.StartNew(m.Template.Path)
		return a, cmd
	}
	return a.routeToCurrent(msg)
}

func (a App) routeToCurrent(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch a.current {
	case common.ScreenDashboard:
		a.dashboard, cmd = a.dashboard.Update(msg)
	case common.ScreenTemplatePicker:
		a.picker, cmd = a.picker.Update(msg)
	}
	return a, cmd
}

func (a App) switchScreen(sw common.SwitchScreenMsg) (tea.Model, tea.Cmd) {
	a.current = sw.Target
	switch sw.Target {
	case common.ScreenTemplatePicker:
		a.picker = templatepicker.New(a.pickerDir, a.pickerCfg)
		return a, a.picker.Init()
	}
	return a, nil
}

// View renders the active screen.
func (a App) View() string {
	switch a.current {
	case common.ScreenDashboard:
		return a.dashboard.View()
	case common.ScreenTemplatePicker:
		return a.picker.View()
	}
	return ""
}
