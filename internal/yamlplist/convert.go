package yamlplist

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/flexphere/lctl/internal/plist"
)

// Options configures context-dependent transformations.
type Options struct {
	// ScriptsDir is the directory lctl resolves slash-less program
	// names to (e.g. `program: backup.sh` under this dir). An empty
	// value disables the program-path shortcut.
	ScriptsDir string
	// Home is used to expand leading `~` in paths. Typically os.UserHomeDir().
	Home string
	// SystemPath is the $PATH expanded for `env` values equal to
	// the literal `$PATH`. Empty disables expansion (the literal is
	// kept as-is).
	SystemPath string
}

// DefaultOptions builds Options from the running process environment.
func DefaultOptions(scriptsDir string) Options {
	home, _ := os.UserHomeDir()
	return Options{
		ScriptsDir: scriptsDir,
		Home:       home,
		SystemPath: os.Getenv("PATH"),
	}
}

// ToAgent converts a user-edited Document into a plist.Agent suitable
// for writing. The two documented shortcuts are applied:
//
//  1. `program: name` (no slash) resolves to ScriptsDir/name.
//  2. `env` values equal to `$PATH` are replaced with opts.SystemPath.
//
// Leading `~` in path-like fields expands to opts.Home. The label is
// automatically scoped to the `lctl.` namespace (see plist.AddLctlPrefix);
// any prefix already present is preserved rather than doubled.
func (d *Document) ToAgent(opts Options) *plist.Agent {
	a := &plist.Agent{
		Label:             plist.AddLctlPrefix(strings.TrimSpace(d.Label)),
		ProgramArguments:  append([]string(nil), d.ProgramArguments...),
		WorkingDirectory:  expandHome(d.WorkingDirectory, opts.Home),
		StandardOutPath:   expandHome(d.Stdout, opts.Home),
		StandardErrorPath: expandHome(d.Stderr, opts.Home),
		Disabled:          d.Disabled,
		RunAtLoad:         d.RunAtLoad,
		StartInterval:     d.Interval,
	}
	if d.Program != "" {
		a.Program = resolveProgram(d.Program, opts)
	}
	if len(d.WatchPaths) > 0 {
		a.WatchPaths = make([]string, 0, len(d.WatchPaths))
		for _, p := range d.WatchPaths {
			a.WatchPaths = append(a.WatchPaths, expandHome(p, opts.Home))
		}
	}
	if len(d.Env) > 0 {
		a.EnvironmentVariables = make(map[string]string, len(d.Env))
		for k, v := range d.Env {
			a.EnvironmentVariables[k] = expandPathValue(v, opts.SystemPath)
		}
	}
	if len(d.Schedule) > 0 {
		entries := make([]plist.CalendarEntry, 0, len(d.Schedule))
		for _, e := range d.Schedule {
			entries = append(entries, plist.CalendarEntry{
				Minute:  e.Minute,
				Hour:    e.Hour,
				Day:     e.Day,
				Weekday: e.Weekday,
				Month:   e.Month,
			})
		}
		a.StartCalendarInterval = entries
	}
	if d.KeepAlive != nil {
		switch {
		case d.KeepAlive.Always != nil:
			a.KeepAlive = *d.KeepAlive.Always
		case d.KeepAlive.Conditions != nil:
			a.KeepAlive = conditionsToPlist(d.KeepAlive.Conditions)
		}
	}
	return a
}

// FromAgent converts a plist.Agent back to a Document. Shortcuts are
// NOT inverted: once expanded, paths stay absolute and `$PATH` stays
// as the resolved PATH string — this keeps round-trip stable. The
// `lctl.` label namespace is stripped so the user sees a bare label;
// ToAgent re-adds it on save.
func FromAgent(a *plist.Agent) *Document {
	d := &Document{
		Label:            plist.StripLctlPrefix(a.Label),
		Program:          a.Program,
		ProgramArguments: append([]string(nil), a.ProgramArguments...),
		WorkingDirectory: a.WorkingDirectory,
		Stdout:           a.StandardOutPath,
		Stderr:           a.StandardErrorPath,
		Disabled:         a.Disabled,
		RunAtLoad:        a.RunAtLoad,
		Interval:         a.StartInterval,
		WatchPaths:       append([]string(nil), a.WatchPaths...),
	}
	if len(a.EnvironmentVariables) > 0 {
		d.Env = make(map[string]string, len(a.EnvironmentVariables))
		for k, v := range a.EnvironmentVariables {
			d.Env[k] = v
		}
	}
	if len(a.StartCalendarInterval) > 0 {
		entries := make([]ScheduleEntry, 0, len(a.StartCalendarInterval))
		for _, e := range a.StartCalendarInterval {
			entries = append(entries, ScheduleEntry{
				Minute:  e.Minute,
				Hour:    e.Hour,
				Day:     e.Day,
				Weekday: e.Weekday,
				Month:   e.Month,
			})
		}
		d.Schedule = entries
	}
	if a.KeepAlive != nil {
		switch k := a.KeepAlive.(type) {
		case bool:
			d.KeepAlive = &KeepAlive{Always: ptrBool(k)}
		case map[string]any:
			d.KeepAlive = &KeepAlive{Conditions: conditionsFromPlist(k)}
		}
	}
	return d
}

func ptrBool(b bool) *bool { return &b }

// expandHome replaces a leading `~` with home. Paths without a leading
// `~` are returned unchanged. An empty string remains empty.
func expandHome(p, home string) string {
	if p == "" || home == "" {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

// expandPathValue replaces the literal `$PATH` with the provided
// systemPath. Any other value passes through.
func expandPathValue(v, systemPath string) string {
	if v == "$PATH" && systemPath != "" {
		return systemPath
	}
	return v
}

// resolveProgram applies the program-path shortcut. If the value
// contains a slash or starts with `~`, treat as a path (with ~ expansion).
// Otherwise treat as a script name and join with ScriptsDir.
func resolveProgram(name string, opts Options) string {
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, "~") || strings.ContainsRune(name, '/') {
		return expandHome(name, opts.Home)
	}
	if opts.ScriptsDir == "" {
		return name
	}
	return filepath.Join(opts.ScriptsDir, name)
}

// conditionsToPlist flattens KeepAliveConditions into the map[string]any
// shape that plist encoding will emit as a dict.
func conditionsToPlist(c *KeepAliveConditions) map[string]any {
	out := map[string]any{}
	if c.SuccessfulExit != nil {
		out["SuccessfulExit"] = *c.SuccessfulExit
	}
	if c.Crashed != nil {
		out["Crashed"] = *c.Crashed
	}
	if c.AfterInitialDemand != nil {
		out["AfterInitialDemand"] = *c.AfterInitialDemand
	}
	if len(c.PathState) > 0 {
		m := map[string]any{}
		for k, v := range c.PathState {
			m[k] = v
		}
		out["PathState"] = m
	}
	if len(c.OtherJobEnabled) > 0 {
		m := map[string]any{}
		for k, v := range c.OtherJobEnabled {
			m[k] = v
		}
		out["OtherJobEnabled"] = m
	}
	return out
}

// conditionsFromPlist rebuilds KeepAliveConditions from a dict that
// came back from plist decode.
func conditionsFromPlist(m map[string]any) *KeepAliveConditions {
	c := &KeepAliveConditions{}
	if v, ok := m["SuccessfulExit"].(bool); ok {
		c.SuccessfulExit = ptrBool(v)
	}
	if v, ok := m["Crashed"].(bool); ok {
		c.Crashed = ptrBool(v)
	}
	if v, ok := m["AfterInitialDemand"].(bool); ok {
		c.AfterInitialDemand = ptrBool(v)
	}
	if v, ok := m["PathState"].(map[string]any); ok {
		c.PathState = map[string]bool{}
		for k, vv := range v {
			if b, ok := vv.(bool); ok {
				c.PathState[k] = b
			}
		}
	}
	if v, ok := m["OtherJobEnabled"].(map[string]any); ok {
		c.OtherJobEnabled = map[string]bool{}
		for k, vv := range v {
			if b, ok := vv.(bool); ok {
				c.OtherJobEnabled[k] = b
			}
		}
	}
	return c
}
