// Package yamlplist converts between lctl's on-screen YAML schema and
// the launchd plist.Agent structure. The schema is lctl-specific
// (snake_case keys) rather than a literal mapping of plist keys, so
// users never see PascalCase and round-trip is structurally stable.
package yamlplist

import (
	"errors"

	"gopkg.in/yaml.v3"
)

// Document is the surface form edited by the user.
type Document struct {
	Label            string            `yaml:"label"`
	Program          string            `yaml:"program,omitempty"`
	ProgramArguments []string          `yaml:"program_arguments,omitempty"`
	WorkingDirectory string            `yaml:"working_directory,omitempty"`
	Stdout           string            `yaml:"stdout,omitempty"`
	Stderr           string            `yaml:"stderr,omitempty"`
	Disabled         bool              `yaml:"disabled,omitempty"`
	RunAtLoad        bool              `yaml:"run_at_load,omitempty"`
	KeepAlive        *KeepAlive        `yaml:"keep_alive,omitempty"`
	Interval         int               `yaml:"interval,omitempty"`
	Schedule         []ScheduleEntry   `yaml:"schedule,omitempty"`
	WatchPaths       []string          `yaml:"watch_paths,omitempty"`
	Env              map[string]string `yaml:"env,omitempty"`
}

// ScheduleEntry corresponds to one StartCalendarInterval dict. Nil
// means "any" (wildcard).
type ScheduleEntry struct {
	Minute  *int `yaml:"minute,omitempty"`
	Hour    *int `yaml:"hour,omitempty"`
	Day     *int `yaml:"day,omitempty"`
	Weekday *int `yaml:"weekday,omitempty"`
	Month   *int `yaml:"month,omitempty"`
}

// KeepAlive is YAML-friendly shim over launchd's bool-or-dict value.
// Exactly one of the inner fields is populated after decode.
type KeepAlive struct {
	Always     *bool
	Conditions *KeepAliveConditions
}

// KeepAliveConditions mirrors the dict form supported by launchd.
type KeepAliveConditions struct {
	SuccessfulExit     *bool           `yaml:"successful_exit,omitempty"`
	Crashed            *bool           `yaml:"crashed,omitempty"`
	PathState          map[string]bool `yaml:"path_state,omitempty"`
	OtherJobEnabled    map[string]bool `yaml:"other_job_enabled,omitempty"`
	AfterInitialDemand *bool           `yaml:"after_initial_demand,omitempty"`
}

// UnmarshalYAML accepts either a bare bool or a dict for keep_alive.
func (k *KeepAlive) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		var b bool
		if err := value.Decode(&b); err != nil {
			return err
		}
		k.Always = &b
		return nil
	}
	if value.Kind == yaml.MappingNode {
		var c KeepAliveConditions
		if err := value.Decode(&c); err != nil {
			return err
		}
		k.Conditions = &c
		return nil
	}
	return errors.New("keep_alive must be a bool or a map")
}

// MarshalYAML emits either a bool or a dict depending on population.
func (k KeepAlive) MarshalYAML() (any, error) {
	if k.Always != nil {
		return *k.Always, nil
	}
	if k.Conditions != nil {
		return k.Conditions, nil
	}
	return nil, nil
}
