# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`lctl` is a macOS terminal UI for managing per-user launchd agents. It wraps
`launchctl` shell-outs in a Bubble Tea dashboard and persists jobs as
`.plist` files under `~/Library/LaunchAgents`. A sibling `lctl cron`
subcommand converts 5-field cron expressions into launchd
`StartCalendarInterval` YAML.

## Common commands

All are thin wrappers over `go`; see `Makefile`.

```
make build           # go build -o bin/lctl ./cmd/lctl
make run             # go run ./cmd/lctl (launches the TUI)
make test            # go test ./...
make test-race       # go test -race ./...
make lint            # golangci-lint run
make check           # vet + lint + test
make fmt             # gofmt -s -w .
```

Single test / package:

```
go test ./internal/tui/dashboard -run TestKeyRefresh -v
go test ./internal/config/...
```

## Architecture

The codebase is a strict **bottom-up layered design**. Each layer knows
only about the ones beneath it; the TUI never shells out directly.

```
cmd/lctl/main.go
   │
   ▼ wires together
internal/tui        (Bubble Tea: App → Dashboard | TemplatePicker screens)
   │
   ▼ calls
internal/service    (JobService.List + Ops: use cases composed from below)
   │         │
   ▼         ▼
internal/plist     internal/launchd
(file I/O +        (launchctl shell-out,
 validation +       parse `list` / `print-disabled`,
 XML codec)         bootstrap/bootout/kickstart/kill)
```

Support packages:

- `internal/config` — XDG paths (`~/.config/lctl/`), `config.toml` loader with
  unknown-key warnings, seeded `defaultConfigTOML`, and keymap/palette
  defaults. `DefaultSettings()` is the canonical source — the seeded
  `config_toml.go` and `~/.config/lctl/config.toml` must stay in sync
  (guarded by `TestSeedMatchesDefaults`).
- `internal/yamlplist` — user-facing YAML schema for agents. `ToAgent`
  expands two shortcuts: `program: name` (no slash) resolves to
  `ScriptsDir/name`, and `env` values equal to `$PATH` are replaced with
  the process PATH.
- `internal/cron` — 5-field cron → `[]plist.CalendarEntry`. Bounded by
  `maxEntries = 288` to prevent plist explosion on wildcard-heavy input.
- `internal/editor` — resolves `$VISUAL` → `$EDITOR` → `vi` and builds
  `exec.Cmd` values for `tea.ExecProcess`.

## Non-obvious invariants

**The `lctl.` label namespace is the scope boundary.** Every agent
lctl creates gets an `lctl.` prefix via `plist.AddLctlPrefix`, and the
dashboard filters `launchctl list` output to that prefix. Agents in
`~/Library/LaunchAgents` not starting with `lctl.` are intentionally
invisible — don't widen this without user direction.

**Edit-save auto-reloads; there is no separate reload key.** After the
user's `$EDITOR` closes on a valid YAML, `dashboard.EditFlow.Finalize`
calls `Bootout` + `Bootstrap` on launchctl so changes take effect.
`service.Ops` deliberately exposes Start/Stop/Restart/Enable/Disable/
Delete but **not** Reload.

**launchctl is accessed through a `Client` interface, never directly.**
Production uses `launchd.Exec` (shells out), tests use the fake in
`internal/launchd/exec_fake_test.go`. The `Kill` / `Bootout` paths
validate labels before shelling out (see `validateLabel`) — preserve
that when adding new launchctl operations.

**Styles flow through `internal/tui/common/theme.go`.** `NewTheme`
takes `config.ListSettings` and `Apply` rebinds package-level
`TitleStyle` / `HelpStyle` / `StatusRunning` / `StatusIdle` /
`StatusErrored` at startup. Don't construct lipgloss styles in
dashboard code — reach for the theme vars so user `config.toml`
palette overrides take effect.

**The dashboard layout is a starship-style format string.** The
default is `$cursor$label$divider$enabled$divider$state$divider$exit$divider$next_run$divider$schedule`
in `config.toml` `[list.layout].template`. The `label` column
auto-stretches to fill terminal width; every other column uses the
configured width. `[list.variables.<name>]` controls per-column
text/color/width, and `[list.variables.<name>.values.<value>]`
overrides per-value (e.g. state's `running` / `errored`). An empty
`color = ""` for a variable is **not** back-filled from defaults — it
intentionally lets value-aware colors take over for `state` / `enabled`.

## Testing patterns

- **No live launchctl in tests.** Package `launchd` uses
  `exec_fake_test.go` to stub shell-outs; `service` and `tui/dashboard`
  tests inject `recordingOps` / `stubLoader` fakes.
- **UI-level tests assert on `Model.View()` strings.** Avoid
  pty/script-based capture — it's flaky. See
  `internal/tui/dashboard/render_test.go`, `stretch_test.go`,
  `values_test.go` for the pattern.
- **Config defaults have a drift test.** When editing
  `config.DefaultSettings`, update `config_toml.go` in the same change;
  `TestSeedMatchesDefaults` will otherwise fail.

## When you change…

- **the default palette/layout/keymap** → update `config.DefaultSettings`
  *and* the commented examples in `config_toml.go`. The seed is what
  new users see on first run.
- **launchctl verbs** → add the method to `launchd.Client` interface,
  implement on `Exec`, stub in `exec_fake_test.go`, then expose through
  `service.Ops` if the TUI needs it.
- **anything in `plist.Agent`** → the YAML codec in `yamlplist` and
  the validator in `plist.Validate` both need matching updates. The
  round-trip test is `yamlplist_test.go`.
