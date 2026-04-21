// Package cron converts 5-field cron expressions into launchd
// StartCalendarInterval entries and computes upcoming fire times.
package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	rcron "github.com/robfig/cron/v3"

	"github.com/flexphere/lctl/internal/plist"
)

// maxEntries caps the combinatorial expansion of a cron expression.
// Highly wildcarded expressions like "*/5 * * * *" produce 12 entries,
// while "0 9 * * 1-5" produces 5; extremely specific expressions can
// explode. We stop well below the point where the resulting plist
// would become unreadable or problematic.
const maxEntries = 288

// Parse parses a 5-field cron expression (minute hour dom month dow).
// It returns StartCalendarInterval entries suitable for a launchd
// agent. Returns an error if the expression is not representable as
// a finite list of calendar dicts or exceeds maxEntries.
func Parse(expr string) ([]plist.CalendarEntry, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields, got %d", len(fields))
	}
	minutes, err := expandField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("minute: %w", err)
	}
	hours, err := expandField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("hour: %w", err)
	}
	days, err := expandField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("day-of-month: %w", err)
	}
	months, err := expandField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("month: %w", err)
	}
	weekdays, err := expandWeekday(fields[4])
	if err != nil {
		return nil, fmt.Errorf("day-of-week: %w", err)
	}

	dayWild := fields[2] == "*"
	weekdayWild := fields[4] == "*"

	var entries []plist.CalendarEntry
	for _, mo := range months {
		for _, m := range minutes {
			for _, h := range hours {
				if !weekdayWild && dayWild {
					// Only weekday constraints apply
					for _, wd := range weekdays {
						entries = append(entries, makeEntry(m, h, nil, monthPtr(mo), intPtr(wd)))
						if len(entries) > maxEntries {
							return nil, fmt.Errorf("cron expression expands to more than %d entries", maxEntries)
						}
					}
					continue
				}
				if !dayWild && weekdayWild {
					// Only day-of-month constraints apply
					for _, d := range days {
						entries = append(entries, makeEntry(m, h, intPtr(d), monthPtr(mo), nil))
						if len(entries) > maxEntries {
							return nil, fmt.Errorf("cron expression expands to more than %d entries", maxEntries)
						}
					}
					continue
				}
				if !dayWild && !weekdayWild {
					// Both constrained: cron semantics is OR. We emit
					// entries for both the day set and the weekday set.
					for _, d := range days {
						entries = append(entries, makeEntry(m, h, intPtr(d), monthPtr(mo), nil))
						if len(entries) > maxEntries {
							return nil, fmt.Errorf("cron expression expands to more than %d entries", maxEntries)
						}
					}
					for _, wd := range weekdays {
						entries = append(entries, makeEntry(m, h, nil, monthPtr(mo), intPtr(wd)))
						if len(entries) > maxEntries {
							return nil, fmt.Errorf("cron expression expands to more than %d entries", maxEntries)
						}
					}
					continue
				}
				// Both wild
				entries = append(entries, makeEntry(m, h, nil, monthPtr(mo), nil))
				if len(entries) > maxEntries {
					return nil, fmt.Errorf("cron expression expands to more than %d entries", maxEntries)
				}
			}
		}
	}
	return entries, nil
}

// monthPtr returns a pointer to mo, or nil if all months are covered
// (month 0 sentinel used by expandField for wildcard).
func monthPtr(mo int) *int {
	if mo == -1 {
		return nil
	}
	return intPtr(mo)
}

func makeEntry(minute, hour int, day, month, weekday *int) plist.CalendarEntry {
	return plist.CalendarEntry{
		Minute:  wildcardToNil(minute),
		Hour:    wildcardToNil(hour),
		Day:     day,
		Month:   month,
		Weekday: weekday,
	}
}

// wildcardToNil returns a pointer to v, or nil when v is the wildcard
// sentinel -1 returned by expandField for "*".
func wildcardToNil(v int) *int {
	if v == -1 {
		return nil
	}
	return intPtr(v)
}

func intPtr(v int) *int { return &v }

// expandField expands a cron field into the set of integer values.
// Supported: "*", "a", "a-b", "a,b,c", "*/n", "a-b/n".
// Wildcard returns a single sentinel value -1.
func expandField(field string, lo, hi int) ([]int, error) {
	if field == "*" {
		return []int{-1}, nil
	}
	seen := map[int]struct{}{}
	parts := strings.Split(field, ",")
	for _, p := range parts {
		values, err := parsePart(p, lo, hi)
		if err != nil {
			return nil, err
		}
		for _, v := range values {
			seen[v] = struct{}{}
		}
	}
	result := make([]int, 0, len(seen))
	for v := lo; v <= hi; v++ {
		if _, ok := seen[v]; ok {
			result = append(result, v)
		}
	}
	return result, nil
}

func parsePart(p string, lo, hi int) ([]int, error) {
	step := 1
	if idx := strings.Index(p, "/"); idx != -1 {
		stepStr := p[idx+1:]
		p = p[:idx]
		n, err := strconv.Atoi(stepStr)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("invalid step %q", stepStr)
		}
		step = n
	}
	var rangeLo, rangeHi int
	switch {
	case p == "*":
		rangeLo = lo
		rangeHi = hi
	case strings.Contains(p, "-"):
		r := strings.SplitN(p, "-", 2)
		a, err := strconv.Atoi(r[0])
		if err != nil {
			return nil, fmt.Errorf("invalid range start %q", r[0])
		}
		b, err := strconv.Atoi(r[1])
		if err != nil {
			return nil, fmt.Errorf("invalid range end %q", r[1])
		}
		rangeLo, rangeHi = a, b
	default:
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid value %q", p)
		}
		if n < lo || n > hi {
			return nil, fmt.Errorf("value %d out of range [%d,%d]", n, lo, hi)
		}
		return []int{n}, nil
	}
	if rangeLo < lo || rangeHi > hi || rangeLo > rangeHi {
		return nil, fmt.Errorf("range %d-%d out of bounds [%d,%d]", rangeLo, rangeHi, lo, hi)
	}
	var out []int
	for v := rangeLo; v <= rangeHi; v += step {
		out = append(out, v)
	}
	return out, nil
}

// expandWeekday mirrors expandField(0,6) but also accepts 7 as Sunday
// and normalizes it to 0 to match launchd semantics.
func expandWeekday(field string) ([]int, error) {
	if field == "*" {
		return []int{-1}, nil
	}
	replaced := strings.ReplaceAll(field, "7", "0")
	values, err := expandField(replaced, 0, 6)
	if err != nil {
		return nil, err
	}
	return values, nil
}

// NextRuns returns up to n future fire times for the given cron
// expression, in local time. The cron parser accepts the standard
// 5-field format (minute hour dom month dow).
func NextRuns(expr string, from time.Time, n int) ([]time.Time, error) {
	parser := rcron.NewParser(rcron.Minute | rcron.Hour | rcron.Dom | rcron.Month | rcron.Dow)
	sched, err := parser.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("parse cron: %w", err)
	}
	out := make([]time.Time, 0, n)
	t := from
	for i := 0; i < n; i++ {
		next := sched.Next(t)
		if next.IsZero() {
			break
		}
		out = append(out, next)
		t = next
	}
	return out, nil
}
