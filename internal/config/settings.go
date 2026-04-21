package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// warnWriter is the sink that LoadSettings uses to report unknown
// config keys. It defaults to os.Stderr so a user running lctl sees
// warnings inline; tests swap it for a buffer via the test helper in
// settings_test.go so they can assert on emitted messages without
// racing against real stderr.
var warnWriter io.Writer = os.Stderr

// Settings is the user-configurable view of lctl's runtime options.
// Fields come from $XDG_CONFIG_HOME/lctl/config.toml, merged over
// DefaultSettings so any omitted key keeps its default.
//
// Top-level sections intentionally avoid a dashboard prefix: everything
// currently configurable belongs to a single screen, and reserving a
// namespace "just in case" another screen appears later adds more
// noise than it saves. If a future screen grows its own settings they
// can live under their own top-level section next to these.
type Settings struct {
	List      ListSettings      `toml:"list"`
	Filter    FilterSettings    `toml:"filter"`
	Templates TemplatesSettings `toml:"templates"`
	Keymap    Keymap            `toml:"keymap"`
}

// TemplatesSettings styles the template-picker screen. Each row
// carries its own Text prefix (e.g. "> " / "  ") and Color so users
// can swap glyphs and palette the same way as the list cursor variable.
type TemplatesSettings struct {
	Selected   TemplateRowSettings `toml:"selected"`
	Unselected TemplateRowSettings `toml:"unselected"`
}

// TemplateRowSettings is the per-row config for the picker. Empty
// Text renders no prefix at all; empty Color leaves the terminal
// default in place.
type TemplateRowSettings struct {
	Text  string `toml:"text"`
	Color string `toml:"color"`
}

// ListSettings groups everything that configures the agent list
// widget: its behavior (auto-refresh), layout (the format template
// and separator), header styling, chrome colors (brand, help,
// semantic status), and per-column rendering.
type ListSettings struct {
	AutoRefresh bool `toml:"auto_refresh"`
	// RefreshInterval is the whole-second cadence of the auto-refresh
	// tick when AutoRefresh is on. Sub-second values aren't supported
	// on purpose: keeping the unit fixed lets the TOML value stay a
	// plain integer instead of a quoted duration string.
	RefreshInterval int                         `toml:"refresh_interval"`
	Layout          LayoutSettings              `toml:"layout"`
	Header          HeaderSettings              `toml:"header"`
	Brand           BrandSettings               `toml:"brand"`
	Help            HelpSettings                `toml:"help"`
	Status          StatusColors                `toml:"status"`
	Variables       map[string]VariableSettings `toml:"variables"`
}

// BrandSettings colors the "lctl" brand string at the top of the
// dashboard. The bold + padding decorations are fixed in code.
type BrandSettings struct {
	Color string `toml:"color"`
}

// HelpSettings colors the dim secondary text: aggregate title,
// footer, help line, op-status messages.
type HelpSettings struct {
	Color string `toml:"color"`
}

// StatusColors is the semantic palette used as per-value defaults
// for $state and $enabled when those variables leave color empty.
// Running is the "active/good" tier, Idle is the neutral grey used
// for loaded / enabled, Errored is the red tier used for errored /
// disabled.
type StatusColors struct {
	Running string `toml:"running"`
	Idle    string `toml:"idle"`
	Errored string `toml:"errored"`
}

// HeaderSettings styles the column-header row rendered directly above
// the list items. Color is a lipgloss-compatible color string; empty
// leaves the terminal default.
type HeaderSettings struct {
	Color string `toml:"color"`
}

// LayoutSettings defines the shared item / header format string and
// the divider substituted for every $divider token. The field is
// named Template rather than Layout so the TOML section [list.layout]
// reads naturally as "list.layout.template = ..." instead of the
// tautological "list.layout.layout = ...".
type LayoutSettings struct {
	Template string `toml:"template"`
	Divider  string `toml:"divider"`
}

// VariableSettings controls how a single format variable is rendered.
//
//   - Text is a small format template applied to the raw value on item
//     rows. Inside the template, both $value and the variable's own
//     name ($label for the label variable, etc.) expand to the raw
//     value; any other tokens are emitted verbatim. Empty text means
//     "use the raw value as-is". Header rows never run this template.
//   - Color is a lipgloss-compatible color string ("241", "red",
//     "#7d56f4"). Empty keeps the compiled-in default: for $state
//     and $enabled that default is value-dependent (running vs
//     errored, enabled vs disabled); for every other variable an
//     empty color means "no color at all". Note this asymmetry —
//     Text and Width treat empty as "use default", but Color's
//     meaning depends on whether a value-specific default exists.
//   - Width is the character width the value is padded or truncated
//     to. Values ≤ 0 are treated as "reset to the compiled-in
//     default"; zero-width rendering is intentionally not supported
//     because every column needs at least one cell worth of room.
//   - Values lets the user style individual possible values of an
//     enumerated variable, e.g. $state's "running" / "errored" or
//     $enabled's "disabled". Entries here take precedence over the
//     top-level Text/Color; entries not listed inherit those as
//     fallbacks, and ultimately the compiled-in value-aware defaults.
type VariableSettings struct {
	Text   string                   `toml:"text"`
	Color  string                   `toml:"color"`
	Width  int                      `toml:"width"`
	Values map[string]ValueSettings `toml:"values"`
}

// ValueSettings overrides Text and Color for a single value of an
// enumerated variable. Empty fields fall through to the parent
// VariableSettings and then to the compiled-in defaults.
type ValueSettings struct {
	Text  string `toml:"text"`
	Color string `toml:"color"`
}

// Keymap maps action names to the string form of the key Bubble Tea
// reports (e.g. "n", "enter", "ctrl+r"). Empty values fall back to
// defaults so users only need to specify overrides.
type Keymap struct {
	New     string `toml:"new"`
	Edit    string `toml:"edit"`
	Log     string `toml:"log"`
	Start   string `toml:"start"`
	Stop    string `toml:"stop"`
	Restart string `toml:"restart"`
	// Toggle flips the selected job's enable/disable state. A single
	// binding is friendlier than separate enable/disable keys because
	// the user's intent is always "flip it" — lctl picks the right
	// direction based on the current launchctl state.
	Toggle  string `toml:"toggle"`
	Delete  string `toml:"delete"`
	Refresh string `toml:"refresh"`
	Quit    string `toml:"quit"`
}

// FilterSettings styles the filter bar drawn above the column header.
// Prompt is the literal string rendered in front of the user's input
// (e.g. "/ " or "> "). Colors accept any lipgloss-compatible value.
// Empty strings leave the terminal default in place.
type FilterSettings struct {
	Prompt      string `toml:"prompt"`
	PromptColor string `toml:"prompt_color"`
	TextColor   string `toml:"text_color"`
}

// DefaultSettings returns the compiled-in defaults. The palette is
// Tokyo Night; auto-refresh is on at a 10-second cadence so the list
// tracks launchd state changes without the user having to hit Ctrl-R.
func DefaultSettings() Settings {
	return Settings{
		List: ListSettings{
			AutoRefresh:     true,
			RefreshInterval: 10,
			Layout: LayoutSettings{
				Template: "$cursor$label$divider$enabled$divider$state$divider$exit$divider$next_run$divider$schedule",
				Divider:  "  ",
			},
			Header: HeaderSettings{Color: "#ffffff"},
			Brand:  BrandSettings{Color: "#bb9af7"},
			Help:   HelpSettings{Color: "#565f89"},
			Status: StatusColors{
				Running: "#9ece6a",
				Idle:    "#565f89",
				Errored: "#f7768e",
			},
			Variables: map[string]VariableSettings{
				"cursor": {
					Text:  "$value",
					Width: 2,
					Values: map[string]ValueSettings{
						"selected":   {Text: "▶ ", Color: "#bb9af7"},
						"unselected": {Text: "  "},
					},
				},
				"label": {Text: "$value", Width: 40},
				"state": {Text: "$value", Width: 14,
					Values: map[string]ValueSettings{
						"running": {Text: "● running", Color: "#9ece6a"},
						"loaded":  {Text: "○ loaded", Color: "#7aa2f7"},
						"errored": {Text: "✗ errored", Color: "#f7768e"},
						"unknown": {Text: "? unknown", Color: "#565f89"},
					},
				},
				"exit":     {Text: "$value", Width: 8},
				"next_run": {Text: "$value", Width: 20},
				"schedule": {Text: "$value", Width: 12},
				"enabled": {Text: "$value", Width: 8,
					Values: map[string]ValueSettings{
						"enabled":  {Text: "on", Color: "#9ece6a"},
						"disabled": {Text: "off", Color: "#f7768e"},
					},
				},
				"divider": {Text: "$value"},
			},
		},
		Filter: FilterSettings{
			Prompt:      "/ ",
			PromptColor: "#bb9af7",
			TextColor:   "#c0caf5",
		},
		Templates: TemplatesSettings{
			Selected:   TemplateRowSettings{Text: "▶ ", Color: "#bb9af7"},
			Unselected: TemplateRowSettings{Text: "  ", Color: "#c0caf5"},
		},
		Keymap: Keymap{
			New:     "n",
			Edit:    "enter",
			Log:     "l",
			Start:   "s",
			Stop:    "x",
			Restart: "r",
			Toggle:  "tab",
			Delete:  "D",
			Refresh: "ctrl+r",
			Quit:    "q",
		},
	}
}

// LoadSettings reads $XDG_CONFIG_HOME/lctl/config.toml on top of
// DefaultSettings. A missing file is not an error. Invalid TOML
// returns the defaults plus the parse error so the caller can surface
// the problem without aborting.
func LoadSettings(paths Paths) (Settings, error) {
	cfg := DefaultSettings()
	path := filepath.Join(paths.Root, "config.toml")
	data, err := os.ReadFile(path) //nolint:gosec // path derived from XDG config root
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}
	md, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	// Warn the user about keys they wrote that no struct field claimed
	// — typos like `[keyamp.dashboard]` or `statys_running = "…"` would
	// otherwise be silently ignored and leave the user puzzled about
	// why their override never took effect. We never fail the load
	// here; unknown keys are informational, not fatal.
	for _, key := range md.Undecoded() {
		_, _ = fmt.Fprintf(warnWriter, "lctl: config: unknown key %q (ignored)\n", key.String())
	}
	// Empty string fields should fall back to defaults. The TOML
	// library leaves them as "" after decode, which would disable
	// actions/colors. Restore any blank slot from defaults.
	fillBlanks(&cfg)
	return cfg, nil
}

// fillBlanks backfills empty strings from defaults so a partial file
// doesn't accidentally null out settings the user didn't touch.
func fillBlanks(c *Settings) {
	defaults := DefaultSettings()
	fillStr := func(dst *string, fallback string) {
		if *dst == "" {
			*dst = fallback
		}
	}
	fillStr(&c.Keymap.New, defaults.Keymap.New)
	fillStr(&c.Keymap.Edit, defaults.Keymap.Edit)
	fillStr(&c.Keymap.Log, defaults.Keymap.Log)
	fillStr(&c.Keymap.Start, defaults.Keymap.Start)
	fillStr(&c.Keymap.Stop, defaults.Keymap.Stop)
	fillStr(&c.Keymap.Restart, defaults.Keymap.Restart)
	fillStr(&c.Keymap.Toggle, defaults.Keymap.Toggle)
	fillStr(&c.Keymap.Delete, defaults.Keymap.Delete)
	fillStr(&c.Keymap.Refresh, defaults.Keymap.Refresh)
	fillStr(&c.Keymap.Quit, defaults.Keymap.Quit)

	if c.List.RefreshInterval <= 0 {
		c.List.RefreshInterval = defaults.List.RefreshInterval
	}
	fillStr(&c.List.Layout.Template, defaults.List.Layout.Template)
	fillStr(&c.List.Layout.Divider, defaults.List.Layout.Divider)
	fillStr(&c.List.Header.Color, defaults.List.Header.Color)
	fillStr(&c.List.Brand.Color, defaults.List.Brand.Color)
	fillStr(&c.List.Help.Color, defaults.List.Help.Color)
	fillStr(&c.List.Status.Running, defaults.List.Status.Running)
	fillStr(&c.List.Status.Idle, defaults.List.Status.Idle)
	fillStr(&c.List.Status.Errored, defaults.List.Status.Errored)
	fillStr(&c.Filter.Prompt, defaults.Filter.Prompt)
	fillStr(&c.Filter.PromptColor, defaults.Filter.PromptColor)
	fillStr(&c.Filter.TextColor, defaults.Filter.TextColor)
	fillStr(&c.Templates.Selected.Text, defaults.Templates.Selected.Text)
	fillStr(&c.Templates.Selected.Color, defaults.Templates.Selected.Color)
	fillStr(&c.Templates.Unselected.Text, defaults.Templates.Unselected.Text)
	fillStr(&c.Templates.Unselected.Color, defaults.Templates.Unselected.Color)
	fillVariables(&c.List.Variables, defaults.List.Variables)
}

// fillVariables merges user-supplied per-variable overrides on top of
// the compiled-in defaults. Missing keys inherit the default entry
// wholesale; present-but-partial entries inherit only the fields they
// left empty. This lets a user write `[list.variables.label]
// color = "red"` without losing the default text/width.
func fillVariables(user *map[string]VariableSettings, defaults map[string]VariableSettings) {
	if *user == nil {
		*user = map[string]VariableSettings{}
	}
	for name, def := range defaults {
		cur, ok := (*user)[name]
		if !ok {
			(*user)[name] = def
			continue
		}
		if cur.Text == "" {
			cur.Text = def.Text
		}
		if cur.Width <= 0 {
			cur.Width = def.Width
		}
		// cur.Color is intentionally NOT backfilled. For $state and
		// $enabled the renderer resolves empty color to the value-
		// specific default (running=green, errored=red, disabled=red,
		// etc.); for other variables it resolves to "no color". Copying
		// `def.Color` in here would paint every row with an explicit
		// color and defeat that value-aware behavior. Keep Color's
		// empty-string escape hatch asymmetric with Text/Width on
		// purpose.
		cur.Values = mergeValues(cur.Values, def.Values)
		(*user)[name] = cur
	}
}

// mergeValues layers user-provided per-value overrides on top of the
// defaults. Missing value keys inherit the whole default entry; present
// entries inherit only their empty Text (Color keeps its asymmetric
// "empty means use the parent-or-semantic default" escape hatch).
func mergeValues(user, defaults map[string]ValueSettings) map[string]ValueSettings {
	if len(defaults) == 0 {
		return user
	}
	if user == nil {
		user = map[string]ValueSettings{}
	}
	for k, def := range defaults {
		cur, ok := user[k]
		if !ok {
			user[k] = def
			continue
		}
		if cur.Text == "" {
			cur.Text = def.Text
		}
		user[k] = cur
	}
	return user
}
