package config

// defaultConfigTOML is the seeded content of ~/.config/lctl/config.toml.
// Every option is commented out with its built-in default so users can
// uncomment the lines they want to change without consulting docs.
const defaultConfigTOML = `# lctl configuration
# Uncomment any line to override the compiled-in default.

[list]
# Periodically refresh the agent list. Defaults to true.
# auto_refresh = true
# refresh_interval is a whole number of seconds; sub-second cadence
# is not supported.
# refresh_interval = 10

[list.layout]
# Shared layout for the list rows and the column header. Each
# $variable expands to the row-appropriate value at render time:
# $label becomes the agent label on rows and "LABEL" on the header,
# $state becomes running / loaded / errored / unknown (or "STATE"),
# etc. $divider is replaced by the 'divider' string below. Unknown
# $variables are printed verbatim so typos stay visible.
# template = "$cursor$label$divider$enabled$divider$state$divider$exit$divider$next_run$divider$schedule"
# divider  = "  "

[list.header]
# Color of the column header row (LABEL / STATE / ...). Empty keeps
# the terminal default.
# color = "#ffffff"

# Per-variable rendering. Each [list.variables.<name>] section can set:
#   text   — format template applied to the raw value on ITEM rows
#            (header rows always show the plain title). Inside the
#            template, $value — or $<name> itself, e.g. $label — is
#            substituted for the raw value; anything else is literal.
#            Empty text means "use the raw value as-is".
#   color  — lipgloss color ("241", "red", "#7d56f4"). Empty does NOT
#            get back-filled from defaults: for $state and $enabled
#            an empty color lets the value-specific default kick in
#            (running=green, errored=red, disabled=red), while for
#            any other variable empty means "no color at all". Set
#            an explicit color if you want to override value-aware
#            coloring.
#   width  — character width the value is padded or truncated to.
#            Width <= 0 means "reset to the compiled-in default";
#            true zero-width rendering is not supported.
# Example — add a "Label:" prefix and paint the column white:
# [list.variables.label]
# text  = "Label: $value"
# color = "#ffffff"
# width = 40

# [list.variables.state]
# text  = "[$value]"
# color = ""      # keep running/loaded/errored per-value coloring
# width = 10

# Per-value overrides for enumerated variables (state / enabled /
# cursor). Each [list.variables.<name>.values.<value>] block can set
# its own text and color; when a field is empty it inherits from
# the parent variable, then from the compiled-in default.
# [list.variables.state.values.running]
# text  = "● running"
# color = "#00ff00"
# [list.variables.state.values.errored]
# text  = "✗ errored"
# color = "#ff0000"
# [list.variables.enabled.values.disabled]
# text  = "off"
# color = "#ff0000"
# The cursor variable has two pseudo-values: "selected" (the row
# under the user's keyboard selection) and "unselected" (every other
# row). Defaults are "▶ " (purple) and "  " so make sure the text is
# two display columns wide — or widen the column with 'width'.
# [list.variables.cursor.values.selected]
# text  = "▶ "
# color = "#bb9af7"
# [list.variables.cursor.values.unselected]
# text  = "  "

# [templates.selected] / [templates.unselected] style the template-
# picker screen opened with the "new" key. 'text' is the prefix
# rendered before each row (swap "> " for "▶ " etc.). 'color' is the
# lipgloss color; leave as "" to keep the terminal default. Selected
# rows are always rendered bold in addition to the color.
# [templates.selected]
# text  = "▶ "
# color = "#bb9af7"
# [templates.unselected]
# text  = "  "
# color = "#c0caf5"

[filter]
# [filter] styles the filter bar drawn above the column header when
# the / key is pressed. 'prompt' is the literal string rendered in
# front of the input ("/ " or "> " are common choices). The colors
# apply to the prompt, the text the user is typing, and the applied
# filter line after Enter. Empty colors fall back to the terminal's
# default palette.
# prompt       = "/ "
# prompt_color = "#bb9af7"
# text_color   = "#c0caf5"

[keymap]
# Use the key strings Bubble Tea reports: "a".."z", "A".."Z",
# "enter", "esc", "tab", "space", or modifier combos like
# "ctrl+r", "ctrl+n", "alt+x".
# new     = "n"
# edit    = "enter"
# log     = "l"
# start   = "s"
# stop    = "x"
# restart = "r"
# toggle  = "tab"      # toggles enable / disable on the selected job
# delete  = "D"
# refresh = "ctrl+r"
# quit    = "q"

[list.brand]
# Title bar color (the "lctl" badge rendered in the top-right).
# Accepts ANSI-256 index ("241"), named color ("red"), or truecolor
# hex ("#bb9af7"). Empty keeps the compiled-in default.
# color = "#bb9af7"

[list.help]
# Color of the footer help line.
# color = "#565f89"

[list.status]
# Semantic palette used as per-value defaults for $state and $enabled
# when their values.<name>.color is left empty. Override per-value
# colors in [list.variables.<name>.values.<value>] if you want a
# different hue for a specific value.
# running = "#9ece6a"
# idle    = "#565f89"
# errored = "#f7768e"
`
