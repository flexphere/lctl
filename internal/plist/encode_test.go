package plist

import (
	"strings"
	"testing"
)

func intPtr(v int) *int { return &v }

func TestEncodeDecodeRoundTrip(t *testing.T) {
	original := &Agent{
		Label:                "com.flexphere.test",
		ProgramArguments:     []string{"/bin/echo", "hello"},
		WorkingDirectory:     "/tmp",
		EnvironmentVariables: map[string]string{"FOO": "bar"},
		StandardOutPath:      "/tmp/out.log",
		StandardErrorPath:    "/tmp/err.log",
		RunAtLoad:            true,
		StartInterval:        300,
		StartCalendarInterval: []CalendarEntry{
			{Hour: intPtr(9), Minute: intPtr(0), Weekday: intPtr(1)},
		},
		WatchPaths: []string{"/tmp/trigger"},
	}
	data, err := EncodeBytes(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "<key>Label</key>") {
		t.Errorf("encoded output missing Label: %s", s)
	}

	decoded, err := DecodeBytes(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Label != original.Label {
		t.Errorf("Label mismatch: got %q want %q", decoded.Label, original.Label)
	}
	if len(decoded.ProgramArguments) != 2 || decoded.ProgramArguments[1] != "hello" {
		t.Errorf("ProgramArguments mismatch: %v", decoded.ProgramArguments)
	}
	if decoded.WorkingDirectory != "/tmp" {
		t.Errorf("WorkingDirectory mismatch: %q", decoded.WorkingDirectory)
	}
	if decoded.EnvironmentVariables["FOO"] != "bar" {
		t.Errorf("EnvironmentVariables mismatch: %v", decoded.EnvironmentVariables)
	}
	if !decoded.RunAtLoad {
		t.Error("RunAtLoad lost")
	}
	if decoded.StartInterval != 300 {
		t.Errorf("StartInterval mismatch: %d", decoded.StartInterval)
	}
	if len(decoded.StartCalendarInterval) != 1 {
		t.Fatalf("StartCalendarInterval mismatch: %+v", decoded.StartCalendarInterval)
	}
	got := decoded.StartCalendarInterval[0]
	if got.Hour == nil || *got.Hour != 9 {
		t.Errorf("Hour mismatch: %+v", got.Hour)
	}
	if got.Weekday == nil || *got.Weekday != 1 {
		t.Errorf("Weekday mismatch: %+v", got.Weekday)
	}
	if len(decoded.WatchPaths) != 1 || decoded.WatchPaths[0] != "/tmp/trigger" {
		t.Errorf("WatchPaths mismatch: %v", decoded.WatchPaths)
	}
}

func TestEncodeOmitsEmpty(t *testing.T) {
	a := &Agent{Label: "com.flexphere.min", Program: "/bin/true"}
	data, err := EncodeBytes(a)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	s := string(data)
	for _, unwanted := range []string{"StartInterval", "WatchPaths", "KeepAlive", "Disabled"} {
		if strings.Contains(s, unwanted) {
			t.Errorf("encoded output should omit %s:\n%s", unwanted, s)
		}
	}
}
