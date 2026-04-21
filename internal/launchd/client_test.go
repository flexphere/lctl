package launchd

import "testing"

func TestStateOf(t *testing.T) {
	pid := 123
	zero := 0
	one := 1
	cases := []struct {
		name string
		e    ListEntry
		want State
	}{
		{"running", ListEntry{Label: "x", PID: &pid}, StateRunning},
		{"loaded_idle", ListEntry{Label: "x", LastExit: &zero}, StateLoaded},
		{"errored", ListEntry{Label: "x", LastExit: &one}, StateErrored},
		{"unknown_exit", ListEntry{Label: "x"}, StateLoaded},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := StateOf(c.e); got != c.want {
				t.Errorf("got %v want %v", got, c.want)
			}
		})
	}
}

func TestTargetAndDomain(t *testing.T) {
	if got := Target(501, "com.flexphere.job"); got != "gui/501/com.flexphere.job" {
		t.Errorf("target mismatch: %q", got)
	}
	if got := Domain(501); got != "gui/501" {
		t.Errorf("domain mismatch: %q", got)
	}
}
