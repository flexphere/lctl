package launchd

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// fakeLaunchctl writes a shell script into a tempdir that mimics a
// subset of launchctl for testing Exec. The script records each
// invocation to $LCTL_CALL_LOG so tests can assert argv.
func fakeLaunchctl(t *testing.T, stdout, stderr string, exit int) (bin, calls string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fake not supported on windows")
	}
	dir := t.TempDir()
	calls = filepath.Join(dir, "calls.log")
	bin = filepath.Join(dir, "launchctl")
	script := "#!/bin/sh\n" +
		"echo \"$@\" >> \"" + calls + "\"\n" +
		"printf %s " + shellQuote(stdout) + "\n" +
		"printf %s " + shellQuote(stderr) + " 1>&2\n" +
		"exit " + itoa(exit) + "\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, calls
}

func shellQuote(s string) string {
	// Single-quote, escape embedded single quotes.
	out := "'"
	for _, r := range s {
		if r == '\'' {
			out += `'\''`
		} else {
			out += string(r)
		}
	}
	out += "'"
	return out
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	// only small ints in tests
	neg := i < 0
	if neg {
		i = -i
	}
	buf := []byte{}
	for i > 0 {
		buf = append([]byte{byte('0' + i%10)}, buf...)
		i /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

func TestExecListParsesFakeOutput(t *testing.T) {
	stdout := "PID\tStatus\tLabel\n" +
		"42\t0\tcom.flexphere.a\n" +
		"-\t1\tcom.flexphere.b\n"
	bin, _ := fakeLaunchctl(t, stdout, "", 0)
	e := &Exec{Bin: bin, UID: 501}
	// Race-detector builds can be slow; give the fake script headroom.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	entries, err := e.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
}

func TestExecBootstrapPassesArgs(t *testing.T) {
	bin, calls := fakeLaunchctl(t, "", "", 0)
	e := &Exec{Bin: bin, UID: 501}
	if err := e.Bootstrap(context.Background(), "/tmp/foo.plist"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	want := "bootstrap gui/501 /tmp/foo.plist\n"
	if got != want {
		t.Errorf("call log mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestExecBootstrapRequiresPath(t *testing.T) {
	bin, _ := fakeLaunchctl(t, "", "", 0)
	e := &Exec{Bin: bin, UID: 501}
	if err := e.Bootstrap(context.Background(), ""); err == nil {
		t.Error("expected error for empty path")
	}
}

func TestExecKickstartArgs(t *testing.T) {
	bin, calls := fakeLaunchctl(t, "", "", 0)
	e := &Exec{Bin: bin, UID: 501}
	if err := e.Kickstart(context.Background(), "com.flexphere.k"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(calls)
	want := "kickstart -k gui/501/com.flexphere.k\n"
	if string(data) != want {
		t.Errorf("call log mismatch:\ngot:  %q\nwant: %q", string(data), want)
	}
}

func TestExecKillDefaultSignal(t *testing.T) {
	bin, calls := fakeLaunchctl(t, "", "", 0)
	e := &Exec{Bin: bin, UID: 501}
	if err := e.Kill(context.Background(), "com.flexphere.k", ""); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(calls)
	want := "kill TERM gui/501/com.flexphere.k\n"
	if string(data) != want {
		t.Errorf("call log mismatch:\ngot:  %q\nwant: %q", string(data), want)
	}
}

func TestExecEnableDisableArgs(t *testing.T) {
	bin, calls := fakeLaunchctl(t, "", "", 0)
	e := &Exec{Bin: bin, UID: 501}
	if err := e.Enable(context.Background(), "com.flexphere.k"); err != nil {
		t.Fatal(err)
	}
	if err := e.Disable(context.Background(), "com.flexphere.k"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(calls)
	want := "enable gui/501/com.flexphere.k\ndisable gui/501/com.flexphere.k\n"
	if string(data) != want {
		t.Errorf("call log mismatch:\ngot:  %q\nwant: %q", string(data), want)
	}
}

func TestExecBootoutIgnoresNotFound(t *testing.T) {
	bin, _ := fakeLaunchctl(t, "", "Could not find service in domain", 113)
	e := &Exec{Bin: bin, UID: 501}
	if err := e.Bootout(context.Background(), "com.flexphere.k"); err != nil {
		t.Errorf("should ignore not-found: %v", err)
	}
}

func TestExecBootoutSurfacesOtherErrors(t *testing.T) {
	bin, _ := fakeLaunchctl(t, "", "real failure", 1)
	e := &Exec{Bin: bin, UID: 501}
	if err := e.Bootout(context.Background(), "com.flexphere.k"); err == nil {
		t.Error("expected error for non-not-found failure")
	}
}

func TestExecRejectsInvalidLabel(t *testing.T) {
	bin, _ := fakeLaunchctl(t, "", "", 0)
	e := &Exec{Bin: bin, UID: 501}
	bad := "com.flexphere; rm -rf /"
	if err := e.Kickstart(context.Background(), bad); err == nil {
		t.Error("expected validation to reject bad label")
	}
	if err := e.Kill(context.Background(), bad, "TERM"); err == nil {
		t.Error("expected validation to reject bad label in kill")
	}
}
