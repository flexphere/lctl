package plist

// Agent is a launchd agent definition corresponding to a .plist file.
// Fields use howett.net/plist tags; zero values are omitted on encode.
type Agent struct {
	Label                 string            `plist:"Label"`
	Program               string            `plist:"Program,omitempty"`
	ProgramArguments      []string          `plist:"ProgramArguments,omitempty"`
	WorkingDirectory      string            `plist:"WorkingDirectory,omitempty"`
	EnvironmentVariables  map[string]string `plist:"EnvironmentVariables,omitempty"`
	StandardOutPath       string            `plist:"StandardOutPath,omitempty"`
	StandardErrorPath     string            `plist:"StandardErrorPath,omitempty"`
	RunAtLoad             bool              `plist:"RunAtLoad,omitempty"`
	KeepAlive             any               `plist:"KeepAlive,omitempty"`
	StartInterval         int               `plist:"StartInterval,omitempty"`
	StartCalendarInterval []CalendarEntry   `plist:"StartCalendarInterval,omitempty"`
	WatchPaths            []string          `plist:"WatchPaths,omitempty"`
	Disabled              bool              `plist:"Disabled,omitempty"`
}

// CalendarEntry corresponds to a single dict inside StartCalendarInterval.
// A nil field means "any" (wildcard).
type CalendarEntry struct {
	Minute  *int `plist:"Minute,omitempty"`
	Hour    *int `plist:"Hour,omitempty"`
	Day     *int `plist:"Day,omitempty"`
	Weekday *int `plist:"Weekday,omitempty"`
	Month   *int `plist:"Month,omitempty"`
}

// KeepAliveActive reports whether KeepAlive resolves to "keep this job
// alive". Both the bool form and the dict form count as active.
func KeepAliveActive(v any) bool {
	switch k := v.(type) {
	case bool:
		return k
	case nil:
		return false
	default:
		// Dict (map[string]any / map[string]bool / etc.) — treat as
		// active since launchd will evaluate conditions at runtime.
		return true
	}
}

// ScheduleKind classifies how the agent is triggered.
type ScheduleKind int

const (
	ScheduleNone ScheduleKind = iota
	ScheduleOnLoad
	ScheduleInterval
	SchedulePeriodic
	ScheduleWatchPath
	ScheduleDaemon
)

// String returns a short label for the schedule kind.
func (k ScheduleKind) String() string {
	switch k {
	case ScheduleOnLoad:
		return "on-load"
	case ScheduleInterval:
		return "interval"
	case SchedulePeriodic:
		return "periodic"
	case ScheduleWatchPath:
		return "watch-path"
	case ScheduleDaemon:
		return "daemon"
	default:
		return "-"
	}
}

// Kind reports the primary schedule kind of the agent. The ordering
// reflects precedence when multiple trigger keys are set, favoring the
// most specific.
func (a *Agent) Kind() ScheduleKind {
	switch {
	case len(a.StartCalendarInterval) > 0:
		return SchedulePeriodic
	case a.StartInterval > 0:
		return ScheduleInterval
	case len(a.WatchPaths) > 0:
		return ScheduleWatchPath
	case KeepAliveActive(a.KeepAlive):
		return ScheduleDaemon
	case a.RunAtLoad:
		return ScheduleOnLoad
	default:
		return ScheduleNone
	}
}
