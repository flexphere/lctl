# Architecture

Developer-oriented overview of how lctl is put together. For the
user-facing config and schema references, see
[`configuration.md`](configuration.md) and
[`yaml-schema.md`](yaml-schema.md).

## Layer diagram

```
                cmd/lctl/main.go
                      │
                      ▼ wires
               internal/tui             Bubble Tea app
           (App → Dashboard | TemplatePicker)
                      │
                      ▼ calls
              internal/service          use cases
        (JobService.List, Ops start/stop/…/delete)
                   │      │
                   ▼      ▼
          internal/plist     internal/launchd
        (file I/O, XML       (launchctl shell-out,
         codec, validate)     domain-scoped verbs)
```

The rule is strict: nothing above the service layer touches
`launchctl` or the filesystem directly, and `launchd`/`plist` know
nothing about the TUI.

## Packages

- **`cmd/lctl/`** — binary entry point. Dispatches between the TUI
  (default) and the `cron` subcommand.
- **`internal/plist/`** — `.plist` file I/O and the `Agent` struct.
  `Store` reads/writes under `~/Library/LaunchAgents`; `scope.go`
  defines the `lctl.` namespace helpers; `validate.go` enforces
  label format and required fields before save.
- **`internal/launchd/`** — `launchctl` wrapper. `Exec` is the
  production client that shells out; `Client` is the interface the
  rest of the code sees, with a fake in `exec_fake_test.go`.
- **`internal/service/`** — composes `plist.Store` and
  `launchd.Client` into two use cases:
  - `JobService.List` merges on-disk plists with `launchctl list`
    runtime state, filtered to the `lctl.` namespace, annotated with
    next-run projections for periodic/interval jobs.
  - `Ops` exposes Start / Stop / Restart / Enable / Disable / Delete.
- **`internal/tui/`** — Bubble Tea `App` and the two screens it
  routes between: `dashboard` (the list view) and `templatepicker`
  (the modal shown by `n`). `common/` holds the theme and
  screen-switch message.
- **`internal/yamlplist/`** — converts between lctl's YAML schema
  (`Document`) and `plist.Agent`. The two documented shortcuts
  (slash-less `program:`, `env: $PATH`) live in `convert.go`.
- **`internal/cron/`** — parses 5-field cron expressions into
  `[]plist.CalendarEntry`. Bounded by `maxEntries = 288` to prevent
  plist bloat on wildcard-heavy input.
- **`internal/editor/`** — resolves `$VISUAL`/`$EDITOR`/`vi` and
  builds an `exec.Cmd` for Bubble Tea's `tea.ExecProcess`.
- **`internal/config/`** — XDG path resolution, `config.toml` loader
  (with unknown-key warnings), `DefaultSettings` as the canonical
  palette/keymap source, and the seeded starter templates.

## Data flow: listing agents

1. User opens the dashboard. `dashboard.Model.Init` schedules a load.
2. `JobService.List` runs in a command:
   - `plist.Store.List()` reads every `*.plist` under
     `~/Library/LaunchAgents`.
   - `launchctl list` and `launchctl print-disabled gui/<uid>` are
     invoked through `launchd.Exec`.
   - Plists outside the `lctl.` namespace (and their parse errors)
     are filtered out.
   - Remaining agents are joined with runtime state and next-run
     time into `service.Job` values.
3. `JobsLoadedMsg` comes back; the dashboard rebuilds its list
   items and renders.
4. If `auto_refresh = true`, a `tea.Tick` at `refresh_interval`
   re-fires the same load.

## Data flow: editing an agent

1. `enter` (or `n` after template selection) → `EditFlow.PrepareEdit`
   or `PrepareNew`:
   - Load (or read template) → convert to YAML → write to a temp
     file.
2. Bubble Tea's `tea.ExecProcess` runs `$EDITOR` on that file.
3. Editor exits → `EditFlow.Finalize`:
   - Read YAML back; parse via `yamlplist.Decode`.
   - Convert to `plist.Agent` with `Document.ToAgent` (applies the
     `program:`-under-scripts and `env: $PATH` shortcuts).
   - Validate via `plist.Validate`; on error, the temp file is
     preserved and the error surfaces in the dashboard status line.
   - Write plist via `plist.Store.Save`.
   - If the label changed, bootout the old registration and delete
     the old plist so `~/Library/LaunchAgents` doesn't accumulate
     orphans.
   - `launchctl bootout` + `bootstrap` the new plist so edits take
     effect immediately. This is why there is no separate "reload"
     keybinding.

## Config loading

`config.LoadSettings` starts from `DefaultSettings()` and overlays
user TOML on top, so omitted fields keep their compiled-in defaults.
After decode:

1. `MetaData.Undecoded()` keys are printed to stderr as warnings.
2. `fillBlanks` backfills empty strings (e.g. `prompt_color = ""`
   becomes the default) and non-positive ints. The `color` field on
   `[list.variables.<name>]` is **not** back-filled — see
   [configuration.md](configuration.md#listvariablesname) for the
   rationale.
3. `fillVariables` merges user-supplied per-variable overrides on top
   of defaults so partial sections don't wipe the rest.

The seeded `~/.config/lctl/config.toml` is generated from
`defaultConfigTOML` in `internal/config/config_toml.go`. A unit test
(`TestSeedMatchesDefaults`) parses the seed and fails if it doesn't
round-trip to `DefaultSettings()`, so the two stay in sync.

## Invariants to preserve

**The `lctl.` label namespace is the scope boundary.** `JobService.List`
skips plists outside it; edits always add the prefix; the dashboard
strips it on display. Don't widen the filter without explicit user
direction.

**All `launchctl` calls go through the `launchd.Client` interface.**
Tests never shell out; production always does. New launchctl verbs
should be added to the interface, implemented on `Exec`, stubbed in
`exec_fake_test.go`, and only then exposed through `service.Ops` or
the TUI.

**Labels are validated before being passed to `launchctl`.** See
`validateLabel` in `internal/launchd/exec.go`. This prevents shell
injection through agent labels.

**UI state is derived, not stored.** The dashboard rebuilds its
`bubbles/list` items from each `JobsLoadedMsg`; there is no separate
cache to keep consistent. `ctrl+r` is just "re-fire the loader".

**Styles flow through `internal/tui/common/theme.go`.** `NewTheme`
takes `config.ListSettings` and `Apply` rebinds package-level vars
(`TitleStyle`, `StatusRunning`, `StatusIdle`, `StatusErrored`, etc.)
at startup. Don't construct `lipgloss.Style` values inline in
screen code — go through the theme so user palette overrides take
effect.

## Adding a new launchctl operation

1. Add the method to `launchd.Client` in `client.go`.
2. Implement it on `launchd.Exec` in `exec.go`. Reuse `validateLabel`
   and the `run` helper so args are shell-safe.
3. Stub it on the fake in `launchd/exec_fake_test.go` and add at
   least one happy-path + one error test in `exec_test.go`.
4. If the TUI needs it, add the method to `service.Ops`, implement
   it on `opService`, and wire a keymap entry in
   `internal/tui/dashboard/dashboard.go`.
5. Update `[keymap]` defaults in `config.DefaultSettings`, the
   commented seed in `config_toml.go`, and the help line in
   `dashboard.helpLine`.

## Adding a new YAML field

1. Add the field to `yamlplist.Document` with the appropriate YAML
   tag.
2. Add the matching field to `plist.Agent` with its plist tag.
3. Wire it through `Document.ToAgent` and the reverse (`FromAgent`)
   so round-trips are stable — `yamlplist_test.go` has the fixture.
4. If it's a trigger (interval/schedule/keep-alive equivalent), update
   `Agent.Kind()` in `plist/model.go` and the `ScheduleKind` enum.
5. If it should show up in the dashboard, add a variable name in
   `dashboard/delegate.go` and document it in
   [configuration.md](configuration.md#listlayout).

## Testing patterns

- **No real `launchctl` in tests.** Inject
  `launchd.Client` / `service.Ops` fakes.
- **TUI tests assert on `Model.View()` strings.** Pty/script capture
  is intentionally avoided — it's flaky across terminals.
- **Config defaults have a drift test.** If you change
  `DefaultSettings`, update `config_toml.go` in the same commit;
  `TestSeedMatchesDefaults` catches mismatches.
- **`go test -race ./...`** should pass; the auto-refresh tick is
  the only concurrency path worth race-checking.

## Why Go / Bubble Tea

- `launchctl` is the hard dependency; macOS-only is fine and Go
  cross-compiles to a single static binary.
- Bubble Tea + bubbles/list give us fzf-feel keyboard navigation, an
  already-working fuzzy filter, and `tea.ExecProcess` for the
  editor hand-off — hard requirements that would each be weeks of
  work in a terminal-drawing library without them.
- `howett.net/plist` handles the XML plist codec reliably; we don't
  hand-format XML.
