package cron

import (
	"testing"
	"time"
)

func TestParseAllWildcardsEmitsNilEntry(t *testing.T) {
	entries, err := Parse("* * * * *")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Minute != nil || e.Hour != nil || e.Day != nil || e.Month != nil || e.Weekday != nil {
		t.Errorf("all-wildcard should produce empty entry: %+v", e)
	}
}

func TestParseDailyAtFixedTime(t *testing.T) {
	entries, err := Parse("0 3 * * *")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Minute == nil || *e.Minute != 0 {
		t.Errorf("minute mismatch: %+v", e.Minute)
	}
	if e.Hour == nil || *e.Hour != 3 {
		t.Errorf("hour mismatch: %+v", e.Hour)
	}
	if e.Day != nil || e.Weekday != nil || e.Month != nil {
		t.Errorf("unexpected non-nil wildcard: %+v", e)
	}
}

func TestParseWeekdayRange(t *testing.T) {
	entries, err := Parse("0 9 * * 1-5")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries (Mon-Fri), got %d", len(entries))
	}
	seen := map[int]bool{}
	for _, e := range entries {
		if e.Weekday == nil {
			t.Fatalf("weekday nil: %+v", e)
		}
		seen[*e.Weekday] = true
		if *e.Hour != 9 || *e.Minute != 0 {
			t.Errorf("entry wrong time: %+v", e)
		}
	}
	for d := 1; d <= 5; d++ {
		if !seen[d] {
			t.Errorf("missing weekday %d", d)
		}
	}
}

func TestParseEveryNMinutes(t *testing.T) {
	entries, err := Parse("*/15 * * * *")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("want 4 entries, got %d", len(entries))
	}
	mins := map[int]bool{}
	for _, e := range entries {
		if e.Minute == nil {
			t.Fatalf("minute nil: %+v", e)
		}
		mins[*e.Minute] = true
		if e.Hour != nil {
			t.Errorf("hour should be nil: %+v", e)
		}
	}
	for _, m := range []int{0, 15, 30, 45} {
		if !mins[m] {
			t.Errorf("missing minute %d", m)
		}
	}
}

func TestParseCommaList(t *testing.T) {
	entries, err := Parse("0 3,15 * * *")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
}

func TestParseBothDayAndWeekday(t *testing.T) {
	// cron semantics: OR between day-of-month and day-of-week
	entries, err := Parse("0 9 1 * 1")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries (day=1 or weekday=1), got %d", len(entries))
	}
}

func TestParseSunday7Normalizes(t *testing.T) {
	entries, err := Parse("0 12 * * 7")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	if entries[0].Weekday == nil || *entries[0].Weekday != 0 {
		t.Errorf("7 should normalize to 0 (Sunday): %+v", entries[0].Weekday)
	}
}

func TestParseInvalidExpressions(t *testing.T) {
	bad := []string{
		"",
		"* * * *",
		"* * * * * *",
		"60 * * * *",
		"-1 * * * *",
		"*/0 * * * *",
		"a * * * *",
		"1-0 * * * *",
	}
	for _, expr := range bad {
		if _, err := Parse(expr); err == nil {
			t.Errorf("expected error for %q", expr)
		}
	}
}

func TestParseMonthConstraint(t *testing.T) {
	entries, err := Parse("0 0 1 1 *")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1, got %d", len(entries))
	}
	e := entries[0]
	if e.Month == nil || *e.Month != 1 {
		t.Errorf("month mismatch: %+v", e.Month)
	}
	if e.Day == nil || *e.Day != 1 {
		t.Errorf("day mismatch: %+v", e.Day)
	}
}

func TestNextRunsDaily(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2026-04-21T10:00:00Z")
	runs, err := NextRuns("0 3 * * *", from, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 3 {
		t.Fatalf("want 3 runs, got %d", len(runs))
	}
	for i := 0; i < len(runs)-1; i++ {
		if !runs[i].Before(runs[i+1]) {
			t.Errorf("runs not ascending: %v", runs)
		}
	}
}

func TestNextRunsInvalidExpr(t *testing.T) {
	if _, err := NextRuns("bad", time.Now(), 3); err == nil {
		t.Error("expected error")
	}
}
