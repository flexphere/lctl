package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flexphere/lctl/internal/launchd"
	"github.com/flexphere/lctl/internal/plist"
)

type fakeStore struct {
	agents []*plist.Agent
	errs   []error
}

func (f *fakeStore) List() ([]*plist.Agent, []error) { return f.agents, f.errs }

type fakeClient struct {
	list     []launchd.ListEntry
	disabled map[string]bool
	err      error
}

func (f *fakeClient) List(context.Context) ([]launchd.ListEntry, error) { return f.list, f.err }
func (f *fakeClient) PrintDisabled(context.Context) (map[string]bool, error) {
	return f.disabled, nil
}
func (f *fakeClient) Bootstrap(context.Context, string) error    { return nil }
func (f *fakeClient) Bootout(context.Context, string) error      { return nil }
func (f *fakeClient) Kickstart(context.Context, string) error    { return nil }
func (f *fakeClient) Kill(context.Context, string, string) error { return nil }
func (f *fakeClient) Enable(context.Context, string) error       { return nil }
func (f *fakeClient) Disable(context.Context, string) error      { return nil }

func intPtr(v int) *int { return &v }

func TestListMergesRuntimeState(t *testing.T) {
	agents := []*plist.Agent{
		{Label: "lctl.com.flexphere.a", Program: "/bin/true", RunAtLoad: true},
		{Label: "lctl.com.flexphere.b", Program: "/bin/true", StartInterval: 60},
	}
	pid := 999
	exitOne := 1
	entries := []launchd.ListEntry{
		{Label: "lctl.com.flexphere.a", PID: &pid},
		{Label: "lctl.com.flexphere.b", LastExit: &exitOne},
	}
	svc := &JobService{
		Store:  &fakeStore{agents: agents},
		Client: &fakeClient{list: entries},
		Now:    func() time.Time { return time.Unix(1700000000, 0) },
	}
	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Jobs) != 2 {
		t.Fatalf("want 2 jobs, got %d", len(got.Jobs))
	}
	byLabel := map[string]Job{}
	for _, j := range got.Jobs {
		byLabel[j.Label] = j
	}
	if byLabel["lctl.com.flexphere.a"].State != launchd.StateRunning {
		t.Errorf("a should be running: %v", byLabel["lctl.com.flexphere.a"].State)
	}
	if byLabel["lctl.com.flexphere.b"].State != launchd.StateErrored {
		t.Errorf("b should be errored: %v", byLabel["lctl.com.flexphere.b"].State)
	}
	if byLabel["lctl.com.flexphere.b"].NextRun == nil {
		t.Error("interval schedule should project next run")
	}
}

func TestListUnknownWhenNotInRuntime(t *testing.T) {
	agents := []*plist.Agent{{Label: "lctl.com.flexphere.only", Program: "/bin/true"}}
	svc := &JobService{Store: &fakeStore{agents: agents}, Client: &fakeClient{}}
	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Jobs) != 1 || got.Jobs[0].State != launchd.StateUnknown {
		t.Errorf("expected unknown state: %+v", got)
	}
}

func TestListLaunchctlError(t *testing.T) {
	svc := &JobService{Store: &fakeStore{}, Client: &fakeClient{err: errors.New("boom")}}
	if _, err := svc.List(context.Background()); err == nil {
		t.Error("expected error")
	}
}

func TestListSurfacesPlistIssues(t *testing.T) {
	store := &fakeStore{
		agents: []*plist.Agent{{Label: "lctl.com.flexphere.ok", Program: "/bin/true"}},
		errs: []error{
			errors.New("lctl.bad.plist: parse failed"), // kept: lctl-prefixed
			errors.New("other.plist: parse failed"),    // filtered out
		},
	}
	svc := &JobService{Store: store, Client: &fakeClient{}}
	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.PlistIssues) != 1 {
		t.Errorf("expected 1 plist issue surfaced, got %d", len(got.PlistIssues))
	}
}

func TestListMergesDisabledFromLaunchctl(t *testing.T) {
	agents := []*plist.Agent{
		{Label: "lctl.com.a", Program: "/bin/true"},
		{Label: "lctl.com.b", Program: "/bin/true"},
	}
	svc := &JobService{
		Store: &fakeStore{agents: agents},
		Client: &fakeClient{disabled: map[string]bool{
			"lctl.com.a": true,
			"lctl.com.b": false,
		}},
	}
	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byLabel := map[string]Job{}
	for _, j := range got.Jobs {
		byLabel[j.Label] = j
	}
	if !byLabel["lctl.com.a"].Disabled {
		t.Error("lctl.com.a should be disabled")
	}
	if byLabel["lctl.com.b"].Disabled {
		t.Error("lctl.com.b should be enabled")
	}
}

func TestListFallsBackToPlistDisabledWhenRuntimeSilent(t *testing.T) {
	agents := []*plist.Agent{
		{Label: "lctl.com.disabled", Program: "/bin/true", Disabled: true},
		{Label: "lctl.com.enabled", Program: "/bin/true"},
	}
	svc := &JobService{
		Store:  &fakeStore{agents: agents},
		Client: &fakeClient{}, // PrintDisabled returns nil map
	}
	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byLabel := map[string]Job{}
	for _, j := range got.Jobs {
		byLabel[j.Label] = j
	}
	if !byLabel["lctl.com.disabled"].Disabled {
		t.Error("disabled plist field should surface when runtime map is empty")
	}
	if byLabel["lctl.com.enabled"].Disabled {
		t.Error("enabled plist should remain enabled")
	}
}

func TestListFiltersNonLctlAgents(t *testing.T) {
	agents := []*plist.Agent{
		{Label: "lctl.com.flexphere.mine", Program: "/bin/true"},
		{Label: "com.apple.trustd.agent", Program: "/usr/libexec/trustd"},
		{Label: "com.google.keystone.agent"},
	}
	svc := &JobService{Store: &fakeStore{agents: agents}, Client: &fakeClient{}}
	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Jobs) != 1 {
		t.Fatalf("want 1 job (lctl-scoped), got %d", len(got.Jobs))
	}
	if got.Jobs[0].Label != "lctl.com.flexphere.mine" {
		t.Errorf("wrong job surfaced: %q", got.Jobs[0].Label)
	}
}

func TestNextRunForPeriodic(t *testing.T) {
	a := &plist.Agent{
		Label: "com.flexphere.p",
		StartCalendarInterval: []plist.CalendarEntry{
			{Minute: intPtr(0), Hour: intPtr(3)},
		},
	}
	from, _ := time.Parse(time.RFC3339, "2026-04-21T00:00:00Z")
	nr := nextRunFor(a, from)
	if nr == nil {
		t.Fatal("expected next run")
	}
	if nr.Hour() != 3 || nr.Minute() != 0 {
		t.Errorf("unexpected next run: %v", nr)
	}
}

func TestNextRunForInterval(t *testing.T) {
	a := &plist.Agent{Label: "x", StartInterval: 120}
	from := time.Unix(1000, 0)
	nr := nextRunFor(a, from)
	if nr == nil {
		t.Fatal("expected next run")
	}
	if nr.Unix() != 1120 {
		t.Errorf("unexpected next run: %v", nr)
	}
}

func TestNextRunForDaemonNil(t *testing.T) {
	a := &plist.Agent{Label: "x", KeepAlive: true}
	if nextRunFor(a, time.Now()) != nil {
		t.Error("daemon should not project next run")
	}
}

func TestNew(t *testing.T) {
	svc := New(&fakeStore{}, &fakeClient{})
	if svc.Store == nil || svc.Client == nil || svc.Now == nil {
		t.Fatal("New must wire all collaborators")
	}
	if svc.Now().IsZero() {
		t.Error("Now should return a valid time")
	}
}

func TestCalendarEntriesToExpr(t *testing.T) {
	expr := calendarEntriesToExpr([]plist.CalendarEntry{
		{Minute: intPtr(0), Hour: intPtr(3)},
	})
	if expr != "0 3 * * *" {
		t.Errorf("unexpected expr: %q", expr)
	}
	if calendarEntriesToExpr(nil) != "" {
		t.Error("empty entries should yield empty")
	}
	if calendarEntriesToExpr([]plist.CalendarEntry{{}, {}}) != "" {
		t.Error("multi entries fallback to empty")
	}
}
