# Agent YAML schema

When you press `n` or `enter` in the dashboard, lctl opens a YAML
document in your `$EDITOR`. On save it is converted to a launchd
`.plist` and written to `~/Library/LaunchAgents/lctl.<label>.plist`.

The YAML schema is lctl-specific (snake_case, flattened) rather than a
literal mapping of launchd's PascalCase plist keys — round-trips are
structurally stable and there's no XML to hand-write.

## Minimal example

```yaml
label: com.example.hello
program: hello.sh
run_at_load: true
```

## Fields

| Key                  | Type                | Maps to launchd                   | Notes |
| -------------------- | ------------------- | --------------------------------- | ----- |
| `label`              | string (required)   | `Label`                           | Reverse-DNS identifier. `lctl.` prefix is added automatically. |
| `program`            | string              | `Program`                         | Absolute path, or a bare name resolved under `~/.config/lctl/scripts/`. |
| `program_arguments`  | list of strings     | `ProgramArguments`                | argv-style list; `program[0]` plus args, or just args if `program` is set. |
| `working_directory`  | string              | `WorkingDirectory`                | Absolute path; `~` expands to `$HOME`. |
| `stdout`             | string              | `StandardOutPath`                 | Log path; `~` expands. |
| `stderr`             | string              | `StandardErrorPath`               | Log path; `~` expands. |
| `disabled`           | bool                | `Disabled`                        | `true` tells launchd to ignore the plist. |
| `run_at_load`        | bool                | `RunAtLoad`                       | Run once when the agent is loaded (login or bootstrap). |
| `keep_alive`         | bool **or** dict    | `KeepAlive`                       | See [KeepAlive](#keep-alive). |
| `interval`           | int (seconds)       | `StartInterval`                   | Run every N seconds. |
| `schedule`           | list of calendar dicts | `StartCalendarInterval`        | See [Schedule](#schedule). |
| `watch_paths`        | list of strings     | `WatchPaths`                      | Trigger when any listed path changes; `~` expands. |
| `env`                | map of strings      | `EnvironmentVariables`            | See [Env](#env). |

Only `label` is required. launchd additionally requires **one** of
`program` or `program_arguments`. All other fields are optional and
omitted from the plist when zero.

## Two shortcuts

### `program:` without a slash

If `program` has no `/` in it, lctl resolves it against
`~/.config/lctl/scripts/`. This lets you drop an executable into that
directory and reference it by bare name:

```yaml
program: backup.sh              # → ~/.config/lctl/scripts/backup.sh
program: /usr/local/bin/brew    # kept as-is
```

### `env: { PATH: $PATH }`

launchd runs agents with a minimal environment (`PATH=/usr/bin:/bin:/usr/sbin:/sbin`).
If an `env` value is the literal string `$PATH`, lctl substitutes the
`$PATH` of the shell running lctl at **save time**, so the plist
captures a stable value:

```yaml
env:
  PATH: $PATH                   # → the concrete PATH of your shell
  GOPATH: /Users/you/go         # literal, unchanged
```

This only applies to the exact string `$PATH`; other references like
`${PATH}` or `$HOME` are not expanded.

## Schedule

Each entry is a dict with any subset of `minute` / `hour` / `day` /
`weekday` / `month` (launchd fires when every set field matches;
omitted fields are wildcards).

```yaml
# Daily at 03:00
schedule:
  - { minute: 0, hour: 3 }

# Weekdays at 09:00 and 17:00
schedule:
  - { minute: 0, hour: 9,  weekday: 1 }
  - { minute: 0, hour: 9,  weekday: 2 }
  - { minute: 0, hour: 9,  weekday: 3 }
  - { minute: 0, hour: 9,  weekday: 4 }
  - { minute: 0, hour: 9,  weekday: 5 }
  - { minute: 0, hour: 17, weekday: 1 }
  # ... etc
```

`weekday` uses launchd's convention: Sunday = 0, Monday = 1, …,
Saturday = 6.

Writing schedule lists by hand is tedious; use `lctl cron` instead:

```sh
lctl cron --inline "0 9,17 * * 1-5"
```

See [cron.md](cron.md).

## Keep alive

Two forms are accepted. The simple bool form keeps the job alive
unconditionally:

```yaml
keep_alive: true
```

The dict form mirrors launchd's conditional restart map:

```yaml
keep_alive:
  successful_exit: false          # restart only on non-zero exit
  crashed: true                   # restart on crash
  path_state:
    /var/run/something.pid: true
  other_job_enabled:
    com.other.upstream: true
  after_initial_demand: true
```

## Env

`env` becomes launchd's `EnvironmentVariables` dict. All values are
strings. The `$PATH` shortcut is the only implicit expansion; anything
else is passed through verbatim:

```yaml
env:
  PATH: $PATH
  HOME: /Users/you
  TZ: Asia/Tokyo
```

## Schedule kinds

The dashboard's `$schedule` column classifies each agent by its
primary trigger. Precedence (most specific wins):

1. `schedule:` → **periodic**
2. `watch_paths:` → **watch-path**
3. `keep_alive:` → **daemon**
4. `interval:` → **interval**
5. `run_at_load:` → **on-load**
6. none of the above → `-`

## Validation

Save fails and the editor reopens if:

- `label` is missing, empty, or not in reverse-DNS form
  (`[A-Za-z0-9_-]+` segments joined by dots, 2+ segments required)
- neither `program` nor `program_arguments` is set
- a path field (`program`, `stdout`, `stderr`, etc.) is not an
  absolute path after `~` expansion (and for `program` is not a
  resolvable scripts-dir name)

Warnings (non-blocking) are printed alongside errors — for example,
setting both `interval` and `schedule` is flagged because only one
takes effect.

## On-disk layout

The plist is written to `~/Library/LaunchAgents/lctl.<label>.plist`
using XML plist (howett.net/plist). PascalCase keys, `<true/>`/`<false/>`
for bools, etc. — standard launchd format. You can inspect the file
after save:

```sh
cat ~/Library/LaunchAgents/lctl.com.example.hello.plist
```
