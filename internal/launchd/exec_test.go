package launchd

import "testing"

func TestValidateLabel(t *testing.T) {
	good := []string{
		"com.flexphere.job",
		"com.flexphere.job_v2",
		"com.flexphere.job-name",
		"A.B.0",
	}
	for _, label := range good {
		if err := validateLabel(label); err != nil {
			t.Errorf("good label %q rejected: %v", label, err)
		}
	}
	bad := []string{
		"",
		"com.flexphere.job; rm -rf /",
		"com.flexphere.job space",
		"com.flexphere/../../etc/passwd",
		"com.flexphere.job\n",
		`com.flexphere."job"`,
		"日本語.label",
	}
	for _, label := range bad {
		if err := validateLabel(label); err == nil {
			t.Errorf("bad label %q accepted", label)
		}
	}
}

func TestIgnoreNotFound(t *testing.T) {
	if err := ignoreNotFound(nil); err != nil {
		t.Error("nil should remain nil")
	}
	if err := ignoreNotFound(dummyErr("Could not find service")); err != nil {
		t.Errorf("not-found should be nil, got %v", err)
	}
	if err := ignoreNotFound(dummyErr("real problem")); err == nil {
		t.Error("real errors must not be swallowed")
	}
}

type dummyErr string

func (d dummyErr) Error() string { return string(d) }
