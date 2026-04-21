// Package service composes the plist store, launchctl client, and
// cron helper into the use cases that the TUI layer consumes.
package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/flexphere/lctl/internal/cron"
	"github.com/flexphere/lctl/internal/launchd"
	"github.com/flexphere/lctl/internal/plist"
)

// Job is the unified view the Dashboard renders per row.
type Job struct {
	Label    string
	State    launchd.State
	PID      *int
	LastExit *int
	Kind     plist.ScheduleKind
	NextRun  *time.Time
	Agent    *plist.Agent
	// Disabled reflects the launchctl enable/disable override for
	// this service. It is independent of State: a disabled job can
	// still appear as "loaded", but launchd will refuse to start it.
	Disabled bool
}

// ListResult bundles the successfully loaded jobs with any per-file
// errors encountered during plist parsing, so the caller can surface
// them in the UI without losing the good rows.
type ListResult struct {
	Jobs        []Job
	PlistIssues []error
}

// Store abstracts plist file I/O so tests can inject fakes.
type Store interface {
	List() ([]*plist.Agent, []error)
}

// JobService lists and operates on user-level launchd agents.
type JobService struct {
	Store  Store
	Client launchd.Client
	// Now is injected for deterministic testing; defaults to time.Now.
	Now func() time.Time
}

// New builds a JobService with real backends.
func New(store Store, client launchd.Client) *JobService {
	return &JobService{Store: store, Client: client, Now: time.Now}
}

// List returns the unified job list, merging on-disk plist definitions
// with runtime state reported by launchctl. Only agents within the
// lctl.* namespace are returned; other plists in ~/Library/LaunchAgents
// (Apple, Homebrew, etc.) are ignored so the Dashboard stays focused
// on jobs lctl itself manages. Parse errors coming from non-lctl plist
// files are similarly filtered out.
func (s *JobService) List(ctx context.Context) (ListResult, error) {
	agents, plistErrs := s.Store.List()
	runtime, err := s.listRuntime(ctx)
	if err != nil {
		return ListResult{PlistIssues: filterLctlIssues(plistErrs)}, err
	}
	// print-disabled is a best-effort read; if it fails (older launchctl
	// or sandboxed environment), we fall back to plist's Disabled field.
	disabled, _ := s.Client.PrintDisabled(ctx)
	now := s.now()
	jobs := make([]Job, 0, len(agents))
	for _, a := range agents {
		if !plist.HasLctlPrefix(a.Label) {
			continue
		}
		j := Job{Label: a.Label, Agent: a, Kind: a.Kind()}
		if rt, ok := runtime[a.Label]; ok {
			j.PID = rt.PID
			j.LastExit = rt.LastExit
			j.State = launchd.StateOf(rt)
		} else {
			j.State = launchd.StateUnknown
		}
		if d, ok := disabled[a.Label]; ok {
			j.Disabled = d
		} else {
			j.Disabled = a.Disabled
		}
		if next := nextRunFor(a, now); next != nil {
			j.NextRun = next
		}
		jobs = append(jobs, j)
	}
	return ListResult{Jobs: jobs, PlistIssues: filterLctlIssues(plistErrs)}, nil
}

// filterLctlIssues keeps only plist-parse errors whose filename begins
// with the lctl prefix. Store.List formats these as
// "<filename>: <detail>", so a string HasPrefix on the error message
// matches the plist we care about.
func filterLctlIssues(errs []error) []error {
	if len(errs) == 0 {
		return nil
	}
	out := make([]error, 0, len(errs))
	for _, e := range errs {
		if strings.HasPrefix(e.Error(), plist.LctlPrefix) {
			out = append(out, e)
		}
	}
	return out
}

func (s *JobService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *JobService) listRuntime(ctx context.Context) (map[string]launchd.ListEntry, error) {
	entries, err := s.Client.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]launchd.ListEntry, len(entries))
	for _, e := range entries {
		out[e.Label] = e
	}
	return out, nil
}

// nextRunFor returns the next fire time for the agent, or nil if the
// schedule cannot be projected (daemons, watch paths, no schedule).
func nextRunFor(a *plist.Agent, from time.Time) *time.Time {
	switch a.Kind() {
	case plist.ScheduleInterval:
		t := from.Add(time.Duration(a.StartInterval) * time.Second)
		return &t
	case plist.SchedulePeriodic:
		expr := calendarEntriesToExpr(a.StartCalendarInterval)
		if expr == "" {
			return nil
		}
		runs, err := cron.NextRuns(expr, from, 1)
		if err != nil || len(runs) == 0 {
			return nil
		}
		return &runs[0]
	default:
		return nil
	}
}

// calendarEntriesToExpr produces the simplest cron expression that the
// first calendar entry represents. Multiple entries with divergent
// fields are not collapsed; the caller falls back to nil in that case.
// Used only for "Next Run" column best-effort display.
func calendarEntriesToExpr(entries []plist.CalendarEntry) string {
	if len(entries) != 1 {
		return ""
	}
	e := entries[0]
	field := func(p *int, wildcard string) string {
		if p == nil {
			return wildcard
		}
		return strconv.Itoa(*p)
	}
	return field(e.Minute, "*") + " " +
		field(e.Hour, "*") + " " +
		field(e.Day, "*") + " " +
		field(e.Month, "*") + " " +
		field(e.Weekday, "*")
}
