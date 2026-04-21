# Configuration

All runtime settings live in `~/.config/lctl/config.toml` (or
`$XDG_CONFIG_HOME/lctl/config.toml`). The file is seeded on first run
with every option commented out and its compiled-in default shown.
Deleting a line or a whole section reverts that value to the default.

Unknown keys print a warning to stderr on startup so typos (like
`[keyamp]` or `status_runing`) don't silently no-op:

```
lctl: config: unknown key "keyamp.new" (ignored)
```

## [list]

Dashboard behavior.

| Key                | Type | Default | Notes                                                                 |
| ------------------ | ---- | ------- | --------------------------------------------------------------------- |
| `auto_refresh`     | bool | `true`  | Poll `launchctl list` on a timer.                                     |
| `refresh_interval` | int  | `10`    | Whole seconds between refreshes. Sub-second cadence is not supported. |

```toml
[list]
auto_refresh     = true
refresh_interval = 10
```

## [list.layout]

Starship-style format string shared by item rows and the column
header. Each `$variable` expands to the row-appropriate value (the
agent's label on a row, `LABEL` on the header, etc.). `$divider` is
substituted for the `divider` string. Unknown `$variables` pass
through verbatim so typos remain visible.

```toml
[list.layout]
template = "$cursor$label$divider$enabled$divider$state$divider$exit$divider$next_run$divider$schedule"
divider  = "  "
```

Available variables:

| Variable    | Row value                                         | Header title |
| ----------- | ------------------------------------------------- | ------------ |
| `$cursor`   | `▶ ` on the selected row, `  ` otherwise          | (pad)        |
| `$label`    | Agent label (stripped of `lctl.` prefix)          | `LABEL`      |
| `$enabled`  | `on` / `off`                                      | `ENABLED`    |
| `$state`    | `running` / `loaded` / `errored` / `unknown`      | `STATE`      |
| `$exit`     | Last exit code, or `-`                            | `EXIT`       |
| `$next_run` | Projected next fire time (periodic/interval jobs) | `NEXT RUN`   |
| `$schedule` | `on-load` / `interval` / `periodic` / `watch-path` / `daemon` | `SCHEDULE` |
| `$divider`  | The `divider` string                              | `divider`    |

The `label` column is the only one that stretches: extra terminal
width is handed to `$label` so the row fills the pane. Other columns
keep their configured width.

## [list.variables.&lt;name&gt;]

Per-column rendering. Each variable can override three keys:

| Key     | Type   | Meaning                                                                 |
| ------- | ------ | ----------------------------------------------------------------------- |
| `text`  | string | Format template applied to item rows. Inside it, `$value` (or `$<name>`, e.g. `$label`) expands to the raw value; anything else is literal. Empty → raw value as-is. Header rows ignore this template. |
| `color` | string | lipgloss color (`"241"`, `"red"`, `"#7D56F4"`). Empty is asymmetric — see below. |
| `width` | int    | Character width for padding/truncation. `0` or negative resets to default. |

**`color = ""` is intentionally asymmetric.** For `$state` and
`$enabled` empty means "use the per-value default" (running green,
errored red, disabled red); for every other variable empty means "no
color". Set an explicit color if you want to override that.

```toml
[list.variables.label]
text  = "Label: $value"
color = "#ffffff"
width = 40
```

### Per-value overrides

Enumerated variables (`state`, `enabled`, `cursor`) accept per-value
subsections at `[list.variables.<name>.values.<value>]` with the same
`text` / `color` keys. A value sub-entry inherits the parent's `text`
if left empty.

```toml
[list.variables.state.values.running]
text  = "● running"
color = "#9ece6a"

[list.variables.state.values.errored]
text  = "✗ errored"
color = "#f7768e"

[list.variables.enabled.values.disabled]
text  = "off"
color = "#f7768e"

[list.variables.cursor.values.selected]
text  = "▶ "
color = "#bb9af7"
```

The `cursor` variable has two pseudo-values: `selected` (row under the
keyboard selection) and `unselected`. Keep both glyphs at two display
columns wide — or widen the column with `width`.

## [list.header]

Column header row styling (the `LABEL ENABLED STATE …` line).

```toml
[list.header]
color = "#ffffff"
```

## [list.brand] / [list.help] / [list.status]

Chrome palette.

```toml
[list.brand]
color = "#bb9af7"       # the "lctl" badge at the top-right

[list.help]
color = "#565f89"       # the footer help line

[list.status]
# Semantic palette used as per-value defaults for $state and $enabled
# when values.<name>.color is empty.
running = "#9ece6a"
idle    = "#565f89"
errored = "#f7768e"
```

## [filter]

The filter bar drawn above the column header when `/` is pressed. The
filter matches against label, state, schedule, and enabled/disabled
all at once, so `/errored`, `/watch-path`, and `/disabled` all work.

```toml
[filter]
prompt       = "/ "
prompt_color = "#bb9af7"
text_color   = "#c0caf5"
```

## [templates]

Styling of the template picker screen (the `n` key).

```toml
[templates.selected]
text  = "▶ "
color = "#bb9af7"

[templates.unselected]
text  = "  "
color = "#c0caf5"
```

Selected rows are always rendered bold in addition to the color.

## [keymap]

Dashboard key bindings. Values are Bubble Tea key strings:
`a`..`z`, `A`..`Z`, `enter`, `esc`, `tab`, `space`, or modifier combos
like `ctrl+r`, `ctrl+n`, `alt+x`. `ctrl+c` always quits as a safety
net even if you remap `quit`.

```toml
[keymap]
new     = "n"
edit    = "enter"
log     = "l"
start   = "s"
stop    = "x"
restart = "r"
toggle  = "tab"     # flips enable/disable on the selected job
delete  = "D"
refresh = "ctrl+r"
quit    = "q"
```

## Color formats

Anywhere a `color` is accepted, three forms work:

- ANSI-256 index as a string: `"241"`
- Named color: `"red"`, `"blue"`, etc.
- Truecolor hex: `"#7D56F4"`

An empty string keeps the compiled-in default **except** for
`[list.variables.<name>].color`, which has the asymmetric behavior
documented above.

## Width semantics

- Positive integer: pad (right-pad with spaces) or truncate (with a
  trailing `…`) to this rune count.
- `0` or negative: reset to the compiled-in default. There is no
  "zero-width render" mode.
- Omitting the key entirely: same as unset — falls back to default.

## When values conflict

Resolution order for `$state` / `$enabled` color:

1. `[list.variables.<var>.values.<value>].color` if non-empty
2. `[list.variables.<var>].color` if non-empty
3. Compiled-in semantic default (`list.status.running` for running,
   `list.status.errored` for errored/disabled, otherwise
   `list.status.idle`)

## Migration notes

- `refresh_interval` used to be a Go duration string (`"10s"`). It is
  now a plain integer number of seconds; writing a string triggers a
  type error on load.
- The `reload` keybinding was removed — edits automatically re-bootstrap
  the plist on save, so a separate reload action is unnecessary.
- `[colors]` was consolidated into `[list.brand]`, `[list.help]`,
  `[list.status]`; `status_stopped` was renamed to `idle`.
