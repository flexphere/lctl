package launchd

import (
	"strings"
	"testing"
)

func TestParseListTypical(t *testing.T) {
	input := "PID\tStatus\tLabel\n" +
		"638\t0\tcom.apple.trustd.agent\n" +
		"-\t0\tcom.apple.mdworker.mail\n" +
		"-\t1\tcom.flexphere.failed\n"
	got, err := parseList(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d entries, want 3", len(got))
	}
	if got[0].Label != "com.apple.trustd.agent" || got[0].PID == nil || *got[0].PID != 638 {
		t.Errorf("entry 0 mismatch: %+v", got[0])
	}
	if got[0].LastExit == nil || *got[0].LastExit != 0 {
		t.Errorf("entry 0 exit mismatch: %+v", got[0].LastExit)
	}
	if got[1].PID != nil {
		t.Errorf("entry 1 should have nil PID: %+v", got[1].PID)
	}
	if got[2].LastExit == nil || *got[2].LastExit != 1 {
		t.Errorf("entry 2 exit mismatch: %+v", got[2])
	}
}

func TestParseListSkipsBlankLines(t *testing.T) {
	input := "PID\tStatus\tLabel\n\n638\t0\tcom.apple.x\n\n"
	got, err := parseList(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
}

func TestParseListMalformedLine(t *testing.T) {
	input := "PID\tStatus\tLabel\ngarbage\n"
	if _, err := parseList(strings.NewReader(input)); err == nil {
		t.Error("expected error for malformed line")
	}
}

func TestParseListBadPID(t *testing.T) {
	input := "abc\t0\tcom.flexphere.x\n"
	if _, err := parseList(strings.NewReader(input)); err == nil {
		t.Error("expected error for non-integer pid")
	}
}

func TestParseListBadStatus(t *testing.T) {
	input := "638\tnotnum\tcom.flexphere.x\n"
	if _, err := parseList(strings.NewReader(input)); err == nil {
		t.Error("expected error for non-integer status")
	}
}
