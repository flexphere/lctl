package common

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/flexphere/lctl/internal/config"
)

// Theme holds the concrete lipgloss styles used across screens. The
// zero value is valid and mirrors the compiled-in defaults so tests
// that don't configure a theme still render correctly.
type Theme struct {
	Title         lipgloss.Style
	Help          lipgloss.Style
	HeaderText    lipgloss.Style
	StatusRunning lipgloss.Style
	StatusIdle    lipgloss.Style
	StatusErrored lipgloss.Style
}

// NewTheme builds a Theme from the list widget's settings. The list
// owns every chrome color the TUI renders today; if another screen
// ever needs its own palette, introduce a sibling theme rather than
// re-routing this one.
func NewTheme(l config.ListSettings) Theme {
	fg := func(v string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(v)) }
	return Theme{
		Title:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(l.Brand.Color)).Padding(0, 1),
		Help:          fg(l.Help.Color),
		HeaderText:    fg(l.Header.Color),
		StatusRunning: fg(l.Status.Running),
		StatusIdle:    fg(l.Status.Idle),
		StatusErrored: fg(l.Status.Errored),
	}
}

// DefaultTheme builds a theme using DefaultSettings — useful for tests
// and for the initial render before config is loaded.
func DefaultTheme() Theme { return NewTheme(config.DefaultSettings().List) }

// TitleStyle / HelpStyle / HeaderText / status variants are package-
// level vars kept for backwards compatibility with existing callers.
// They are rebound via Apply() at startup once user config is read.
var (
	TitleStyle    = DefaultTheme().Title
	HelpStyle     = DefaultTheme().Help
	HeaderText    = DefaultTheme().HeaderText
	StatusRunning = DefaultTheme().StatusRunning
	StatusIdle    = DefaultTheme().StatusIdle
	StatusErrored = DefaultTheme().StatusErrored
)

// Apply rebinds the package-level styles so screens that rely on them
// pick up user customization. Call once from main after LoadSettings.
func Apply(t Theme) {
	TitleStyle = t.Title
	HelpStyle = t.Help
	HeaderText = t.HeaderText
	StatusRunning = t.StatusRunning
	StatusIdle = t.StatusIdle
	StatusErrored = t.StatusErrored
}
