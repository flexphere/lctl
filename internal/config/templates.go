package config

// defaultTemplates is the set of starter YAML files copied into
// $XDG_CONFIG_HOME/lctl/templates the first time lctl runs.
//
// Each file is documented with `#` comments that explain the purpose
// of the template and offer sensible sample values. Note that these
// comments are discarded when the YAML is converted to .plist on save.
//
// lctl scopes all managed jobs under the `lctl.` label namespace: the
// label you write here is saved as `lctl.<your-label>` on disk and
// shown stripped in the dashboard / when you reopen the YAML.
var defaultTemplates = map[string]string{
	"periodic.yaml": `# Periodic template — runs on a cron-like schedule.
# Replace <name>, adjust the schedule entries, then save to register.
# The saved plist label will be automatically namespaced as
# "lctl.<label>" — you do not need to type the prefix here.

label: com.example.periodic
# 'program' without a slash resolves to ~/.config/lctl/scripts/<name>.
# Use an absolute path to reference an arbitrary binary.
program: example.sh

# Use 'lctl cron "<expr>"' to convert a cron expression into this block.
schedule:
  - { minute: 0, hour: 3 }

# Log paths. Omit to skip. Paths beginning with ~ expand to $HOME.
stdout: ~/Library/Logs/com.example.periodic.out.log
stderr: ~/Library/Logs/com.example.periodic.err.log

# launchd runs with a minimal environment by default (PATH=/usr/bin:/bin:/usr/sbin:/sbin).
# Use '$PATH' as a placeholder to inject the PATH from the shell running lctl at save time.
env:
  PATH: $PATH
`,

	"on-login.yaml": `# On-login template — runs once when the agent is loaded.

label: com.example.on-login
program: example.sh

run_at_load: true

stdout: ~/Library/Logs/com.example.on-login.out.log
stderr: ~/Library/Logs/com.example.on-login.err.log

env:
  PATH: $PATH
`,

	"interval.yaml": `# Interval template — runs every N seconds.

label: com.example.interval
program: example.sh

# Run every 5 minutes.
interval: 300

stdout: ~/Library/Logs/com.example.interval.out.log
stderr: ~/Library/Logs/com.example.interval.err.log

env:
  PATH: $PATH
`,

	"watch-path.yaml": `# Watch-path template — triggers when the given file or directory changes.

label: com.example.watch
program: example.sh

watch_paths:
  - ~/Documents/trigger

stdout: ~/Library/Logs/com.example.watch.out.log
stderr: ~/Library/Logs/com.example.watch.err.log

env:
  PATH: $PATH
`,

	"daemon.yaml": `# Daemon template — long-lived process, restarted on exit.

label: com.example.daemon
program: example.sh

run_at_load: true

# Simple bool form: always keep alive.
keep_alive: true

# Or the dict form with conditional restart rules:
# keep_alive:
#   successful_exit: false      # restart only when exit was non-zero
#   crashed: true               # restart on crash
#   path_state:
#     /var/run/something.pid: true
#   other_job_enabled:
#     com.other.upstream: true
#   after_initial_demand: true

stdout: ~/Library/Logs/com.example.daemon.out.log
stderr: ~/Library/Logs/com.example.daemon.err.log

env:
  PATH: $PATH
`,

	"blank.yaml": `# Blank template — fill in the fields you need.
#
# Reference:
#   label: reverse-DNS identifier (required)
#   program: path to executable (slash-less → resolved under scripts/)
#   program_arguments: list form with argv
#   working_directory: optional cwd for the job
#   stdout / stderr: log file paths (~ expansion supported)
#   disabled: bool; when true, launchctl ignores the plist
#   run_at_load: bool; launchd runs once on load
#   keep_alive: bool or dict (see daemon.yaml)
#   interval: StartInterval in seconds
#   schedule: list of calendar dicts {minute,hour,day,weekday,month}
#   watch_paths: list of files/dirs to monitor
#   env: environment variables map

label: com.example.blank
program: example.sh
`,
}
