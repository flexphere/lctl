// Package templatepicker presents the YAML template files found in
// $XDG_CONFIG_HOME/lctl/templates so the user can pick a scaffold to
// edit when creating a new agent.
package templatepicker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/flexphere/lctl/internal/config"
	"github.com/flexphere/lctl/internal/tui/common"
)

// Template describes one *.yaml file on disk.
type Template struct {
	Name string // file name (e.g. "periodic.yaml")
	Path string // absolute path
}

// TemplateChosenMsg is emitted when the user picks a template with Enter.
type TemplateChosenMsg struct {
	Template Template
}

// Model is the TemplatePicker screen.
type Model struct {
	dir       string
	templates []Template
	err       error
	cursor    int
	styles    styles
}

// styles holds the lipgloss styles + prefix markers used to render
// the picker's rows. Selected rows are rendered bold so they stand
// out even when the user omits a color override.
type styles struct {
	selectedText    string
	selectedStyle   lipgloss.Style
	unselectedText  string
	unselectedStyle lipgloss.Style
}

func newStyles(s config.TemplatesSettings) styles {
	return styles{
		selectedText:    s.Selected.Text,
		selectedStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(s.Selected.Color)),
		unselectedText:  s.Unselected.Text,
		unselectedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color(s.Unselected.Color)),
	}
}

// New returns a picker rooted at dir. The directory is read lazily in
// Init so callers can reuse a zero-value Model across navigations.
// `cfg` supplies the selected / unselected row colors; an empty value
// falls back to the compiled-in defaults via fillBlanks upstream.
func New(dir string, cfg config.TemplatesSettings) Model {
	return Model{dir: dir, styles: newStyles(cfg)}
}

// Init loads the template list.
func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return loadedMsg{templates: list(m.dir)} }
}

// loadedMsg internally delivers the filesystem read result.
type loadedMsg struct {
	templates []Template
	err       error
}

// list reads the directory and returns an alphabetically sorted list
// of .yaml / .yml files.
func list(dir string) []Template {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Template
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		out = append(out, Template{Name: name, Path: filepath.Join(dir, name)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Update advances model state.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadedMsg:
		m.templates = msg.templates
		m.err = msg.err
		if m.cursor >= len(m.templates) {
			m.cursor = 0
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, func() tea.Msg { return common.SwitchScreenMsg{Target: common.ScreenDashboard} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.templates)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.templates) == 0 {
				return m, nil
			}
			chosen := m.templates[m.cursor]
			return m, func() tea.Msg { return TemplateChosenMsg{Template: chosen} }
		}
	}
	return m, nil
}

// View renders the picker.
func (m Model) View() string {
	title := common.TitleStyle.Render("lctl — new agent / pick a template")
	help := common.HelpStyle.Render("↑/↓ or j/k:navigate  Enter:open editor  Esc:back  q:quit")
	if m.err != nil {
		return lipgloss.JoinVertical(lipgloss.Left, title,
			common.StatusErrored.Render(fmt.Sprintf("template dir error: %v", m.err)), help)
	}
	if len(m.templates) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, title,
			common.HelpStyle.Render(fmt.Sprintf("no *.yaml files in %s", m.dir)), help)
	}
	lines := []string{title}
	for i, t := range m.templates {
		marker := m.styles.unselectedText
		style := m.styles.unselectedStyle
		if i == m.cursor {
			marker = m.styles.selectedText
			style = m.styles.selectedStyle
		}
		lines = append(lines, style.Render(marker+t.Name))
	}
	lines = append(lines, help)
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
