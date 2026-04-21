package main

import (
	"bytes"
	"strings"
	"testing"
)

func runCron(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var so, se bytes.Buffer
	code = cronSubcommand(args, &so, &se)
	return so.String(), se.String(), code
}

func TestCronSimpleExpression(t *testing.T) {
	out, _, code := runCron(t, "0 3 * * *")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "schedule:") {
		t.Errorf("missing schedule header: %q", out)
	}
	if !strings.Contains(out, "minute: 0") || !strings.Contains(out, "hour: 3") {
		t.Errorf("entries missing: %q", out)
	}
}

func TestCronWeekdayRangeProducesMultipleEntries(t *testing.T) {
	out, _, code := runCron(t, "0 9 * * 1-5")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	lines := strings.Count(out, "- {")
	if lines != 5 {
		t.Errorf("want 5 entries, got %d:\n%s", lines, out)
	}
}

func TestCronInlineOmitsHeader(t *testing.T) {
	out, _, code := runCron(t, "--inline", "0 3 * * *")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(out, "schedule:") {
		t.Errorf("--inline must not emit schedule header: %q", out)
	}
	if strings.HasPrefix(out, "  ") {
		t.Errorf("--inline must not indent: %q", out)
	}
	if !strings.Contains(out, "- {") {
		t.Errorf("entries missing: %q", out)
	}
}

func TestCronInvalidExpressionReturnsNonZero(t *testing.T) {
	_, stderr, code := runCron(t, "bogus")
	if code == 0 {
		t.Error("expected non-zero exit")
	}
	if !strings.Contains(stderr, "cron") {
		t.Errorf("expected error message, got %q", stderr)
	}
}

func TestCronMissingArg(t *testing.T) {
	_, stderr, code := runCron(t)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr, "Usage") {
		t.Errorf("expected usage, got %q", stderr)
	}
}
