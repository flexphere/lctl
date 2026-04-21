package dashboard

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/flexphere/lctl/internal/config"
	"github.com/flexphere/lctl/internal/launchd"
	"github.com/flexphere/lctl/internal/plist"
	"github.com/flexphere/lctl/internal/service"
	"github.com/flexphere/lctl/internal/tui/common"
)

// cursorPad is the fixed-width cursor slot used when a row is not the
// current selection. Matches the visible width of "❯ " so headers and
// rows align regardless of which one is drawn.
const cursorPad = "  "

// Fallback widths used when the user left a variable's width unset.
// These match the original hardcoded layout so existing deployments
// render identically without touching their config.toml.
const (
	defaultLabelWidth    = 40
	defaultStateWidth    = 14
	defaultExitWidth     = 8
	defaultNextRunWidth  = 20
	defaultScheduleWidth = 12
	defaultEnabledWidth  = 8
	defaultCursorWidth   = 2
)

// jobItem adapts a service.Job to bubbles/list's Item interface.
// `label` keeps the full lctl-scoped identifier for downstream ops;
// `display` is the prefix-stripped form shown in the UI.
type jobItem struct {
	label    string
	display  string
	state    launchd.State
	disabled bool
	lastExit string
	nextRun  string
	schedule string
}

// FilterValue is what list's fuzzy filter matches against. We include
// every visible column so `/error`, `/calendar`, or `/disabled` works
// the same way a label query does.
func (i jobItem) FilterValue() string {
	return strings.Join([]string{
		i.display,
		i.state.String(),
		i.schedule,
		enabledLabel(i.disabled),
	}, " ")
}

// rowRenderer carries the format configuration so the delegate and
// the header-line helper can share the same rendering pipeline. It
// is built once in Model.New and reused across draws.
type rowRenderer struct {
	tokens    []Token
	divider   string
	variables map[string]config.VariableSettings
	// textTokens caches the parsed text template per variable so we
	// don't re-parse on every row.
	textTokens map[string][]Token
	// valueTextTokens holds per-value text templates for enumerated
	// variables, keyed as [variable][rawValue].
	valueTextTokens map[string]map[string][]Token
	// naturalWidth is the sum of every column's configured width
	// plus the visible width of each literal/divider token. Used to
	// decide how much surplus to hand to the stretch variable when
	// the terminal is wider than the layout.
	naturalWidth int
	// stretchVar is the variable whose width is expanded to absorb
	// any extra terminal width, so the list occupies the full row.
	stretchVar string
	// termWidth is the last-known terminal width. Zero disables
	// stretching (useful for unit tests that skip WindowSizeMsg).
	termWidth int
}

// newRowRenderer parses the layout and text templates up front. The
// parent `text` for each variable and every entry in `values.<name>`
// is pre-parsed so render time is a straight map lookup.
func newRowRenderer(f config.LayoutSettings, vars map[string]config.VariableSettings) *rowRenderer {
	tt := make(map[string][]Token, len(vars))
	vt := make(map[string]map[string][]Token, len(vars))
	for name, v := range vars {
		if v.Text == "" {
			tt[name] = []Token{{Kind: TokVar, VarName: "value"}}
		} else {
			tt[name] = ParseFormat(v.Text)
		}
		if len(v.Values) == 0 {
			continue
		}
		per := make(map[string][]Token, len(v.Values))
		for rawValue, entry := range v.Values {
			if entry.Text == "" {
				continue
			}
			per[rawValue] = ParseFormat(entry.Text)
		}
		if len(per) > 0 {
			vt[name] = per
		}
	}
	r := &rowRenderer{
		tokens:          ParseFormat(f.Template),
		divider:         f.Divider,
		variables:       vars,
		textTokens:      tt,
		valueTextTokens: vt,
		stretchVar:      "label",
	}
	r.naturalWidth = r.computeNaturalWidth()
	return r
}

// computeNaturalWidth sums each layout token's contribution, using
// configured widths for variables (never the stretched width) and
// visible widths for literals and the divider. Run once after the
// layout is parsed.
func (r *rowRenderer) computeNaturalWidth() int {
	total := 0
	for _, t := range r.tokens {
		switch t.Kind {
		case TokLiteral:
			total += lipgloss.Width(t.Literal)
		case TokVar:
			if t.VarName == "divider" {
				total += lipgloss.Width(r.divider)
				continue
			}
			total += r.configuredWidth(t.VarName)
		}
	}
	return total
}

// SetTerminalWidth records the latest terminal width so widthOf can
// stretch the nominated variable to make the list fill the row.
func (r *rowRenderer) SetTerminalWidth(w int) { r.termWidth = w }

// delegate renders one jobItem per line by expanding the shared layout
// template.
type delegate struct {
	renderer *rowRenderer
}

// Height is the number of screen rows a single item consumes.
func (delegate) Height() int { return 1 }

// Spacing is blank rows between items. Zero = fzf-like density.
func (delegate) Spacing() int { return 0 }

// Update is unused; operations are handled one layer up in Model.
func (delegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render writes the item line to w by expanding the layout template.
func (d delegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(jobItem)
	if !ok {
		return
	}
	selected := index == m.Index()
	line := d.renderer.renderItem(it, selected)
	if selected {
		line = selectedRowStyle.Render(line)
	}
	_, _ = fmt.Fprint(w, line)
}

// renderItem expands the layout once for a single job row. Each
// variable is routed through styledValue so per-value text and color
// overrides work uniformly for every variable, including $cursor.
func (r *rowRenderer) renderItem(it jobItem, selected bool) string {
	plain := lipgloss.NewStyle()
	vars := map[string]string{
		"cursor":   r.styledValue("cursor", cursorValue(selected), plain),
		"label":    r.styledValue("label", it.display, plain),
		"state":    r.styledValue("state", it.state.String(), stateStyle(it.state)),
		"exit":     r.styledValue("exit", it.lastExit, plain),
		"next_run": r.styledValue("next_run", it.nextRun, plain),
		"schedule": r.styledValue("schedule", it.schedule, plain),
		"enabled":  r.styledValue("enabled", enabledLabel(it.disabled), enabledStyle(it.disabled)),
		"divider":  r.styledValue("divider", r.divider, plain),
	}
	return RenderFormat(r.tokens, vars)
}

// styledValue is the single render path for a variable cell. It looks
// up per-value text first, falls back to the parent text template,
// then pads to the configured width and applies a color in this
// priority order: values.<raw>.color → variable.color → fallback.
func (r *rowRenderer) styledValue(name, raw string, fallback lipgloss.Style) string {
	text := r.expandValueText(name, raw)
	padded := padOrTrim(text, r.widthOf(name))
	if c := r.valueColor(name, raw); c != "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(padded)
	}
	return fallback.Render(padded)
}

// renderHeader expands the layout with header titles substituted for
// each variable. Text templates are skipped on header rows so the
// titles stay predictable (e.g. "Label: LABEL" is avoided).
func (r *rowRenderer) renderHeader() string {
	vars := map[string]string{
		"cursor":   r.headerValue("cursor", cursorPad),
		"label":    r.headerValue("label", "LABEL"),
		"state":    r.headerValue("state", "STATE"),
		"exit":     r.headerValue("exit", "EXIT"),
		"next_run": r.headerValue("next_run", "NEXT RUN"),
		"schedule": r.headerValue("schedule", "SCHEDULE"),
		"enabled":  r.headerValue("enabled", "ENABLED"),
		"divider":  r.headerValue("divider", r.divider),
	}
	return RenderFormat(r.tokens, vars)
}

// headerValue renders a variable cell for the column-header row. Text
// templates are deliberately skipped so titles stay literal (otherwise
// "LABEL" would render as "Label: LABEL" when a text template is set);
// per-value color overrides do not apply on the header either.
func (r *rowRenderer) headerValue(name, raw string) string {
	padded := padOrTrim(raw, r.widthOf(name))
	if v, ok := r.variables[name]; ok && v.Color != "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(v.Color)).Render(padded)
	}
	return padded
}

// expandValueText applies the text template chosen for `raw`, using
// the per-value template when present and falling back to the parent
// variable template when not.
func (r *rowRenderer) expandValueText(variable, raw string) string {
	tokens, ok := r.valueTextTokens[variable][raw]
	if !ok {
		tokens, ok = r.textTokens[variable]
		if !ok {
			return raw
		}
	}
	return RenderFormat(tokens, map[string]string{"value": raw, variable: raw})
}

// valueColor returns the color chosen for `raw`, with the resolution
// order: values.<raw>.color → variable.color → "" (caller applies a
// compiled-in default).
func (r *rowRenderer) valueColor(variable, raw string) string {
	v, ok := r.variables[variable]
	if !ok {
		return ""
	}
	if entry, ok := v.Values[raw]; ok && entry.Color != "" {
		return entry.Color
	}
	return v.Color
}

// widthOf returns the effective column width, adding any stretch
// surplus when this variable is the designated stretch target and the
// terminal is wider than the layout would otherwise fill.
func (r *rowRenderer) widthOf(name string) int {
	base := r.configuredWidth(name)
	if name == r.stretchVar && r.termWidth > 0 && r.naturalWidth > 0 {
		if extra := r.termWidth - r.naturalWidth; extra > 0 {
			return base + extra
		}
	}
	return base
}

// configuredWidth returns the base width a variable requests through
// config, or the compiled-in default when the user left it unset. It
// ignores the stretch state so it is safe to use while computing
// naturalWidth.
func (r *rowRenderer) configuredWidth(name string) int {
	if v, ok := r.variables[name]; ok && v.Width > 0 {
		return v.Width
	}
	switch name {
	case "cursor":
		return defaultCursorWidth
	case "label":
		return defaultLabelWidth
	case "state":
		return defaultStateWidth
	case "exit":
		return defaultExitWidth
	case "next_run":
		return defaultNextRunWidth
	case "schedule":
		return defaultScheduleWidth
	case "enabled":
		return defaultEnabledWidth
	default:
		return 0 // divider and unknowns keep their natural width
	}
}

// cursorValue returns a semantic token for the current row selection:
// "selected" or "unselected". The actual character is resolved via
// the $cursor variable's per-value text template so users can change
// the glyph (e.g. "▶ " / "> ") from config.
func cursorValue(selected bool) string {
	if selected {
		return "selected"
	}
	return "unselected"
}

// selectedRowStyle is a subtle full-line highlight; keep it light so
// the state color still dominates.
var selectedRowStyle = lipgloss.NewStyle().Bold(true)

// stateStyle maps a runtime state to the matching common theme style.
func stateStyle(s launchd.State) lipgloss.Style {
	switch s {
	case launchd.StateRunning:
		return common.StatusRunning
	case launchd.StateErrored:
		return common.StatusErrored
	default:
		return common.StatusIdle
	}
}

// enabledStyle dims the disabled value so it stands out as "not the
// default" without hijacking attention from the runtime state column.
func enabledStyle(disabled bool) lipgloss.Style {
	if disabled {
		return common.StatusErrored
	}
	return common.StatusIdle
}

// enabledLabel renders the boolean as a short human-readable token.
func enabledLabel(disabled bool) string {
	if disabled {
		return "disabled"
	}
	return "enabled"
}

// padOrTrim rightpads s to width with spaces, or truncates (with a
// trailing ellipsis) when s is too long. Width is counted in runes so
// multi-byte labels align correctly. Width ≤ 0 returns s unchanged.
func padOrTrim(s string, width int) string {
	if width <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) > width {
		if width <= 1 {
			return string(r[:width])
		}
		return string(r[:width-1]) + "…"
	}
	return s + strings.Repeat(" ", width-len(r))
}

// toItems converts service jobs into list items with display-stripped
// labels.
func toItems(jobs []service.Job) []list.Item {
	items := make([]list.Item, 0, len(jobs))
	for _, j := range jobs {
		items = append(items, jobItem{
			label:    j.Label,
			display:  plist.StripLctlPrefix(j.Label),
			state:    j.State,
			disabled: j.Disabled,
			lastExit: formatExit(j.LastExit),
			nextRun:  formatNext(j.NextRun),
			schedule: j.Kind.String(),
		})
	}
	return items
}

func formatExit(p *int) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *p)
}

func formatNext(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04")
}
