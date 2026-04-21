# `lctl cron`

Converts a 5-field cron expression into the `schedule:` YAML block
used by lctl agents (which in turn maps to launchd's
`StartCalendarInterval`).

launchd doesn't accept cron syntax natively — every schedule has to
be a list of `{minute, hour, day, weekday, month}` dicts. This
subcommand does that expansion for you.

## Usage

```sh
lctl cron "<minute> <hour> <day> <month> <weekday>"
lctl cron --inline "<expr>"
```

Without `--inline` the output includes the `schedule:` header so you
can append it to a YAML file as-is:

```sh
$ lctl cron "0 9 * * 1-5"
schedule:
  - { minute: 0, hour: 9, weekday: 1 }
  - { minute: 0, hour: 9, weekday: 2 }
  - { minute: 0, hour: 9, weekday: 3 }
  - { minute: 0, hour: 9, weekday: 4 }
  - { minute: 0, hour: 9, weekday: 5 }
```

With `--inline`, only the list entries are emitted so you can paste
under an existing `schedule:` key:

```sh
$ lctl cron --inline "0 9 * * 1-5"
  - { minute: 0, hour: 9, weekday: 1 }
  - { minute: 0, hour: 9, weekday: 2 }
  - { minute: 0, hour: 9, weekday: 3 }
  - { minute: 0, hour: 9, weekday: 4 }
  - { minute: 0, hour: 9, weekday: 5 }
```

## Supported syntax

Each of the five fields accepts:

| Form     | Meaning                                  | Example         |
| -------- | ---------------------------------------- | --------------- |
| `*`      | Every value                              | `* * * * *`     |
| `N`      | Exactly N                                | `0 3 * * *`     |
| `N,M,…`  | List of values                           | `0 9,12,17 * * *` |
| `A-B`    | Range                                    | `0 9 * * 1-5`   |
| `*/N`    | Every N starting from the field's min    | `*/15 * * * *`  |
| `A-B/N`  | Stepped range                            | `0 9-17/2 * * *`|

The weekday field additionally accepts 3-letter names (`sun`, `mon`,
`tue`, `wed`, `thu`, `fri`, `sat`) and Sunday is both `0` and `7` for
convenience. launchd internally uses Sunday = 0.

## Combining `day` and `weekday`

cron treats day-of-month and day-of-week as **OR** when both are
constrained. `lctl cron` mirrors that by emitting one entry per
day-of-month *and* one entry per weekday:

```sh
$ lctl cron --inline "0 9 15 * 1"
  - { minute: 0, hour: 9, day: 15 }
  - { minute: 0, hour: 9, weekday: 1 }
```

This runs at 09:00 on the 15th of every month **and** every Monday.

## Examples

```sh
# Every 5 minutes
lctl cron "*/5 * * * *"

# Daily at 03:00
lctl cron "0 3 * * *"

# Weekdays at 09:00 and 17:00
lctl cron "0 9,17 * * 1-5"

# First of each month, noon
lctl cron "0 12 1 * *"

# Every 2 hours during office hours, weekdays
lctl cron "0 9-17/2 * * 1-5"
```

## Limits

Wildcard-heavy expressions like `* * * * *` expand combinatorially.
`lctl cron` caps the output at **288 entries** (corresponds to
5-minute granularity across a day) — beyond that, the resulting plist
is impractical to edit or reason about and the command errors out:

```
$ lctl cron "*/1 * * * *"
cron expression expands to more than 288 entries
```

Pick a coarser granularity, or use `interval:` seconds when you truly
want a tight cadence.
