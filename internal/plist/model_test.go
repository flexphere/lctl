package plist

import "testing"

func TestAgentKind(t *testing.T) {
	cases := []struct {
		name  string
		agent Agent
		want  ScheduleKind
	}{
		{"none", Agent{Label: "a"}, ScheduleNone},
		{"onload", Agent{Label: "a", RunAtLoad: true}, ScheduleOnLoad},
		{"daemon", Agent{Label: "a", KeepAlive: true}, ScheduleDaemon},
		{"interval", Agent{Label: "a", StartInterval: 60}, ScheduleInterval},
		{"periodic", Agent{Label: "a", StartCalendarInterval: []CalendarEntry{{}}}, SchedulePeriodic},
		{"watch", Agent{Label: "a", WatchPaths: []string{"/tmp"}}, ScheduleWatchPath},
		{"periodic_wins", Agent{Label: "a", RunAtLoad: true, StartCalendarInterval: []CalendarEntry{{}}}, SchedulePeriodic},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.agent.Kind(); got != c.want {
				t.Errorf("got %v want %v", got, c.want)
			}
		})
	}
}

func TestScheduleKindString(t *testing.T) {
	if ScheduleNone.String() != "-" {
		t.Errorf("ScheduleNone string mismatch")
	}
	if SchedulePeriodic.String() != "periodic" {
		t.Errorf("SchedulePeriodic string mismatch")
	}
}
