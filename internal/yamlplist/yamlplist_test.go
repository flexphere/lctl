package yamlplist

import (
	"strings"
	"testing"

	"github.com/flexphere/lctl/internal/plist"
)

func intPtr(v int) *int { return &v }

func opts() Options {
	return Options{
		ScriptsDir: "/Users/x/.config/lctl/scripts",
		Home:       "/Users/x",
		SystemPath: "/opt/homebrew/bin:/usr/bin",
	}
}

func TestDecodeRejectsUnknownFields(t *testing.T) {
	_, err := Decode([]byte("label: com.x\ntypo_key: oops\n"))
	if err == nil {
		t.Error("expected error on unknown field")
	}
}

func TestDecodeEncodeRoundtripSimple(t *testing.T) {
	src := `label: com.flexphere.x
program: /bin/true
run_at_load: true
schedule:
  - {minute: 0, hour: 3}
env:
  PATH: $PATH
`
	d, err := Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if d.Label != "com.flexphere.x" || d.Program != "/bin/true" {
		t.Errorf("decode mismatch: %+v", d)
	}
	if !d.RunAtLoad {
		t.Error("run_at_load lost")
	}
	if len(d.Schedule) != 1 || *d.Schedule[0].Minute != 0 || *d.Schedule[0].Hour != 3 {
		t.Errorf("schedule mismatch: %+v", d.Schedule)
	}
	if d.Env["PATH"] != "$PATH" {
		t.Errorf("env literal not preserved: %+v", d.Env)
	}
	out, err := Encode(d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "label: com.flexphere.x") {
		t.Errorf("roundtrip: %s", out)
	}
}

func TestToAgentProgramShortcut(t *testing.T) {
	d := &Document{Label: "com.x", Program: "run.sh"}
	a := d.ToAgent(opts())
	if a.Program != "/Users/x/.config/lctl/scripts/run.sh" {
		t.Errorf("program shortcut not applied: %q", a.Program)
	}
}

func TestToAgentProgramAbsoluteUnchanged(t *testing.T) {
	d := &Document{Label: "com.x", Program: "/usr/bin/env"}
	a := d.ToAgent(opts())
	if a.Program != "/usr/bin/env" {
		t.Errorf("absolute path should pass through: %q", a.Program)
	}
}

func TestToAgentTildeExpansion(t *testing.T) {
	d := &Document{
		Label:            "com.x",
		Program:          "~/bin/tool",
		Stdout:           "~/Library/Logs/x.log",
		WorkingDirectory: "~",
	}
	a := d.ToAgent(opts())
	if a.Program != "/Users/x/bin/tool" {
		t.Errorf("~ not expanded in program: %q", a.Program)
	}
	if a.StandardOutPath != "/Users/x/Library/Logs/x.log" {
		t.Errorf("~ not expanded in stdout: %q", a.StandardOutPath)
	}
	if a.WorkingDirectory != "/Users/x" {
		t.Errorf("~ bare not expanded: %q", a.WorkingDirectory)
	}
}

func TestToAgentPathShortcutExpanded(t *testing.T) {
	d := &Document{
		Label:   "com.x",
		Program: "/bin/true",
		Env:     map[string]string{"PATH": "$PATH", "FOO": "bar"},
	}
	a := d.ToAgent(opts())
	if a.EnvironmentVariables["PATH"] != "/opt/homebrew/bin:/usr/bin" {
		t.Errorf("$PATH not expanded: %q", a.EnvironmentVariables["PATH"])
	}
	if a.EnvironmentVariables["FOO"] != "bar" {
		t.Errorf("other env passed through: %q", a.EnvironmentVariables["FOO"])
	}
}

func TestToAgentSchedule(t *testing.T) {
	d := &Document{
		Label:   "com.x",
		Program: "/bin/true",
		Schedule: []ScheduleEntry{
			{Minute: intPtr(0), Hour: intPtr(9), Weekday: intPtr(1)},
		},
	}
	a := d.ToAgent(opts())
	if len(a.StartCalendarInterval) != 1 {
		t.Fatalf("schedule count mismatch")
	}
	if *a.StartCalendarInterval[0].Hour != 9 {
		t.Errorf("hour mismatch: %v", a.StartCalendarInterval[0].Hour)
	}
}

func TestToAgentKeepAliveBool(t *testing.T) {
	tru := true
	d := &Document{Label: "com.x", Program: "/bin/true", KeepAlive: &KeepAlive{Always: &tru}}
	a := d.ToAgent(opts())
	if a.KeepAlive != true {
		t.Errorf("keep_alive bool not set: %+v", a.KeepAlive)
	}
}

func TestToAgentKeepAliveDict(t *testing.T) {
	fals := false
	tru := true
	d := &Document{
		Label:   "com.x",
		Program: "/bin/true",
		KeepAlive: &KeepAlive{Conditions: &KeepAliveConditions{
			SuccessfulExit: &fals,
			Crashed:        &tru,
			PathState:      map[string]bool{"/var/run/x.pid": true},
		}},
	}
	a := d.ToAgent(opts())
	m, ok := a.KeepAlive.(map[string]any)
	if !ok {
		t.Fatalf("keep_alive should be map, got %T", a.KeepAlive)
	}
	if m["SuccessfulExit"] != false {
		t.Errorf("SuccessfulExit: %v", m["SuccessfulExit"])
	}
	if m["Crashed"] != true {
		t.Errorf("Crashed: %v", m["Crashed"])
	}
	ps, ok := m["PathState"].(map[string]any)
	if !ok || ps["/var/run/x.pid"] != true {
		t.Errorf("PathState: %v", m["PathState"])
	}
}

func TestKeepAliveUnmarshalBool(t *testing.T) {
	d, err := Decode([]byte("label: com.x\nkeep_alive: true\n"))
	if err != nil {
		t.Fatal(err)
	}
	if d.KeepAlive == nil || d.KeepAlive.Always == nil || !*d.KeepAlive.Always {
		t.Errorf("bool keep_alive not parsed: %+v", d.KeepAlive)
	}
}

func TestKeepAliveUnmarshalDict(t *testing.T) {
	src := `label: com.x
keep_alive:
  successful_exit: false
  crashed: true
  path_state:
    /tmp/x.pid: true
`
	d, err := Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if d.KeepAlive == nil || d.KeepAlive.Conditions == nil {
		t.Fatalf("conditions missing: %+v", d.KeepAlive)
	}
	c := d.KeepAlive.Conditions
	if c.SuccessfulExit == nil || *c.SuccessfulExit {
		t.Errorf("SuccessfulExit mismatch: %+v", c.SuccessfulExit)
	}
	if c.Crashed == nil || !*c.Crashed {
		t.Errorf("Crashed mismatch: %+v", c.Crashed)
	}
	if c.PathState["/tmp/x.pid"] != true {
		t.Errorf("path_state: %v", c.PathState)
	}
}

func TestFromAgentRoundtrip(t *testing.T) {
	a := &plist.Agent{
		Label:                "com.x",
		Program:              "/bin/true",
		WorkingDirectory:     "/tmp",
		StandardOutPath:      "/tmp/out.log",
		StandardErrorPath:    "/tmp/err.log",
		RunAtLoad:            true,
		StartInterval:        60,
		EnvironmentVariables: map[string]string{"FOO": "bar"},
		WatchPaths:           []string{"/tmp/watch"},
		StartCalendarInterval: []plist.CalendarEntry{
			{Minute: intPtr(0), Hour: intPtr(3)},
		},
		KeepAlive: map[string]any{"SuccessfulExit": false},
	}
	d := FromAgent(a)
	if d.Label != "com.x" || d.Program != "/bin/true" {
		t.Errorf("basic fields mismatch: %+v", d)
	}
	if d.KeepAlive == nil || d.KeepAlive.Conditions == nil {
		t.Fatalf("keep_alive dict lost: %+v", d.KeepAlive)
	}
	if d.KeepAlive.Conditions.SuccessfulExit == nil || *d.KeepAlive.Conditions.SuccessfulExit {
		t.Errorf("SuccessfulExit mismatch")
	}
	if len(d.Schedule) != 1 || *d.Schedule[0].Minute != 0 {
		t.Errorf("schedule: %+v", d.Schedule)
	}
}

func TestFromAgentKeepAliveBool(t *testing.T) {
	a := &plist.Agent{Label: "com.x", KeepAlive: true}
	d := FromAgent(a)
	if d.KeepAlive == nil || d.KeepAlive.Always == nil || !*d.KeepAlive.Always {
		t.Errorf("bool roundtrip: %+v", d.KeepAlive)
	}
}

func TestDefaultOptionsPullsHomeAndPath(t *testing.T) {
	t.Setenv("PATH", "/opt/custom/bin")
	o := DefaultOptions("/tmp/scripts")
	if o.ScriptsDir != "/tmp/scripts" {
		t.Errorf("scripts dir: %q", o.ScriptsDir)
	}
	if o.SystemPath != "/opt/custom/bin" {
		t.Errorf("path: %q", o.SystemPath)
	}
	if o.Home == "" {
		t.Error("home should populate")
	}
}
