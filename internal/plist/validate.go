package plist

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// labelPattern matches a reverse-DNS style label: segments of
// [A-Za-z0-9_-] separated by dots, 2+ segments required.
var labelPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+)+$`)

// ValidationIssue represents a single validation problem or warning.
type ValidationIssue struct {
	Field   string
	Message string
	Warning bool // true = non-blocking hint, false = blocking error
}

// Error returns a formatted issue string.
func (i ValidationIssue) Error() string {
	kind := "error"
	if i.Warning {
		kind = "warn"
	}
	return fmt.Sprintf("%s: %s: %s", kind, i.Field, i.Message)
}

// ValidationResult aggregates issues.
type ValidationResult struct {
	Issues []ValidationIssue
}

// HasErrors reports whether any non-warning issue is present.
func (r ValidationResult) HasErrors() bool {
	for _, i := range r.Issues {
		if !i.Warning {
			return true
		}
	}
	return false
}

// Err returns a combined error if any blocking issue exists.
func (r ValidationResult) Err() error {
	if !r.HasErrors() {
		return nil
	}
	var msgs []string
	for _, i := range r.Issues {
		if !i.Warning {
			msgs = append(msgs, i.Error())
		}
	}
	return errors.New(strings.Join(msgs, "; "))
}

// Validate checks the agent for structural errors and surfaces warnings
// (e.g. TCC-sensitive paths). It does not touch launchctl.
func Validate(a *Agent) ValidationResult {
	var out ValidationResult

	add := func(field, msg string, warn bool) {
		out.Issues = append(out.Issues, ValidationIssue{Field: field, Message: msg, Warning: warn})
	}

	if strings.TrimSpace(a.Label) == "" {
		add("Label", "must not be empty", false)
	} else if !labelPattern.MatchString(a.Label) {
		add("Label", "should be reverse-DNS (e.g. com.flexphere.job)", false)
	}

	if a.Program == "" && len(a.ProgramArguments) == 0 {
		add("Program", "either Program or ProgramArguments is required", false)
	}

	if a.Program != "" {
		validateExecPath(&out, "Program", a.Program)
	}
	if len(a.ProgramArguments) > 0 {
		validateExecPath(&out, "ProgramArguments[0]", a.ProgramArguments[0])
	}

	if a.WorkingDirectory != "" && !filepath.IsAbs(a.WorkingDirectory) {
		add("WorkingDirectory", "must be absolute", false)
	}

	for _, field := range []struct {
		name string
		val  string
	}{
		{"StandardOutPath", a.StandardOutPath},
		{"StandardErrorPath", a.StandardErrorPath},
	} {
		if field.val == "" {
			continue
		}
		if !filepath.IsAbs(field.val) {
			add(field.name, "must be absolute", false)
		}
	}

	// TCC warnings: paths that touch sensitive dirs may require Full Disk Access.
	tccWarn(&out, a)

	kinds := 0
	if len(a.StartCalendarInterval) > 0 {
		kinds++
	}
	if a.StartInterval > 0 {
		kinds++
	}
	if len(a.WatchPaths) > 0 {
		kinds++
	}
	if KeepAliveActive(a.KeepAlive) {
		kinds++
	}
	if kinds == 0 && !a.RunAtLoad {
		add("Schedule", "no trigger defined (RunAtLoad/KeepAlive/StartInterval/StartCalendarInterval/WatchPaths)", true)
	}

	return out
}

func validateExecPath(out *ValidationResult, field, path string) {
	if !filepath.IsAbs(path) {
		out.Issues = append(out.Issues, ValidationIssue{Field: field, Message: "must be absolute path", Warning: false})
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		out.Issues = append(out.Issues, ValidationIssue{Field: field, Message: fmt.Sprintf("stat failed: %v", err), Warning: true})
		return
	}
	if info.IsDir() {
		out.Issues = append(out.Issues, ValidationIssue{Field: field, Message: "points to a directory", Warning: false})
		return
	}
	if info.Mode().Perm()&0o111 == 0 {
		out.Issues = append(out.Issues, ValidationIssue{Field: field, Message: "not executable", Warning: true})
	}
}

// tccSensitivePrefixes lists home-relative directory names that often
// require Full Disk Access when accessed by a launchd agent.
var tccSensitivePrefixes = []string{
	"Library/Mail",
	"Library/Messages",
	"Library/Safari",
	"Library/Calendars",
	"Library/AddressBook",
	"Documents",
	"Desktop",
	"Downloads",
	"Pictures",
}

func tccWarn(out *ValidationResult, a *Agent) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	check := func(field, value string) {
		if value == "" {
			return
		}
		abs := filepath.Clean(value)
		for _, prefix := range tccSensitivePrefixes {
			sensitive := filepath.Join(home, prefix)
			if abs == sensitive || strings.HasPrefix(abs, sensitive+string(filepath.Separator)) {
				out.Issues = append(out.Issues, ValidationIssue{
					Field:   field,
					Message: fmt.Sprintf("path under %s may require Full Disk Access (TCC)", prefix),
					Warning: true,
				})
				return
			}
		}
	}
	check("WorkingDirectory", a.WorkingDirectory)
	check("StandardOutPath", a.StandardOutPath)
	check("StandardErrorPath", a.StandardErrorPath)
	for _, p := range a.WatchPaths {
		check("WatchPaths", p)
	}
}
