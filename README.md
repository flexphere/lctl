# lctl

A terminal UI for managing macOS `launchd` user agents — list the jobs
you own, edit them in your `$EDITOR` as readable YAML, and let `lctl`
handle the plist round-trip and `launchctl bootstrap`/`bootout` for you.

```
  lctl
  LABEL                                     ENABLED   STATE           EXIT      NEXT RUN              SCHEDULE
▶ com.example.backup                        on        ● running       -         2026-04-21 15:00      periodic
  com.example.sync                          on        ○ loaded        0         -                     on-load
  com.example.watch                         off       ✗ errored       127       -                     watch-path

  n:new  enter:edit  l:log  s/x/r:start/stop/restart  tab:toggle  D:delete  /:filter  ctrl+r:refresh  q:quit
```

## Why

`launchctl` is powerful but the UX ends at the command line: every
operation is a sub-command, plists are XML, and there is no list view
of "the jobs I care about" — everything in `~/Library/LaunchAgents`
shows up together. `lctl` scopes itself to a single namespace
(`lctl.*`) and gives you:

- a single-pane dashboard with live runtime state (running / loaded /
  errored, last exit, next scheduled run)
- agent YAML in your editor of choice — no XML, no PascalCase
- `lctl cron "*/5 9-17 * * 1-5"` to convert cron expressions into
  launchd `StartCalendarInterval` blocks
- per-agent start/stop/restart/enable/disable/delete, with
  `launchctl bootstrap` fired automatically on save

## Requirements

- macOS (only platform that ships `launchctl`)
- Go 1.24+ to build from source

## Install

From source:

```sh
git clone https://github.com/flexphere/lctl
cd lctl
make build              # → bin/lctl
cp bin/lctl /usr/local/bin/     # or anywhere on $PATH
```

Run directly without installing:

```sh
make run                # go run ./cmd/lctl
```

## Quickstart

1. **Launch** — `lctl`. On first run it creates
   `~/.config/lctl/config.toml`, `~/.config/lctl/templates/`, and
   `~/.config/lctl/scripts/` with starter files.
2. **Create an agent** — press `n`, pick a template (periodic,
   on-login, interval, watch-path, daemon, blank), hit `enter`. Your
   `$EDITOR` opens on the YAML; save and quit. `lctl` validates,
   writes the plist under `~/Library/LaunchAgents/lctl.<label>.plist`,
   and bootstraps it into launchd.
3. **Operate** — select a row and press `s` / `x` / `r` to
   start / stop / restart, `tab` to toggle enable/disable, `l` to tail
   the agent's stdout log, `enter` to re-edit, `D` to delete.
4. **Schedule from cron** — outside the TUI, run
   `lctl cron "0 3 * * *"` to print a `schedule:` block you can paste
   into a template.

## Key bindings (default)

| Key      | Action                              |
| -------- | ----------------------------------- |
| `n`      | New agent (template picker)         |
| `enter`  | Edit selected agent in `$EDITOR`    |
| `l`      | Open the agent's stdout log         |
| `s`      | Start (kickstart)                   |
| `x`      | Stop (SIGTERM)                      |
| `r`      | Restart (stop + kickstart)          |
| `tab`    | Toggle enable/disable               |
| `D`      | Delete agent + plist                |
| `/`      | Filter by label / state / schedule  |
| `ctrl+r` | Refresh list                        |
| `q`      | Quit (`ctrl+c` always quits too)    |

Rebind anything in `[keymap]` in `config.toml`.

## Config layout

```
~/.config/lctl/
├── config.toml          runtime settings (keymap, palette, layout)
├── templates/           starter YAML files shown by the `n` picker
│   ├── periodic.yaml
│   ├── on-login.yaml
│   ├── interval.yaml
│   ├── watch-path.yaml
│   ├── daemon.yaml
│   └── blank.yaml
└── scripts/             target for slash-less `program:` names
```

Drop your own YAML files into `templates/` to have them appear in the
`n` picker. Put executables you want to reference by bare name into
`scripts/` and write `program: my-script.sh` in the YAML.

## The `lctl.` namespace

`lctl` only manages agents whose label starts with `lctl.`. You write
`label: com.example.backup` in the YAML; on save lctl stores it as
`lctl.com.example.backup` and lists it as `com.example.backup` in the
dashboard. Other plists in `~/Library/LaunchAgents` (Homebrew, Apple,
your own pre-lctl ones) are intentionally invisible — that separation
is the whole point.

## Documentation

- [`docs/configuration.md`](docs/configuration.md) — full `config.toml`
  reference: keymap, palette, layout/format, filter, templates.
- [`docs/yaml-schema.md`](docs/yaml-schema.md) — agent YAML schema with
  every supported field.
- [`docs/cron.md`](docs/cron.md) — the `lctl cron` subcommand.
- [`docs/architecture.md`](docs/architecture.md) — developer overview
  of the layered design.

## Building & testing

```sh
make check              # vet + lint + test (the CI-ready bundle)
make test               # go test ./...
make test-race          # go test -race ./...
make lint               # golangci-lint run
```

## License

See `LICENSE` if present in the repository root.
