package main

import (
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/flexphere/lctl/internal/cron"
	"github.com/flexphere/lctl/internal/plist"
)

// cronSubcommand converts a 5-field cron expression into the
// lctl-native `schedule:` YAML block.
//
// Usage:
//
//	lctl cron [--inline] "<cron-expression>"
//
// Without --inline the output includes the `schedule:` header; with
// --inline only the list entries are emitted so callers can paste
// under an existing key.
func cronSubcommand(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cron", flag.ContinueOnError)
	fs.SetOutput(stderr)
	inline := fs.Bool("inline", false, "emit only the list entries, without the 'schedule:' header")
	fs.Usage = func() {
		_, _ = fmt.Fprintln(stderr, `Usage: lctl cron [--inline] "<minute hour day month weekday>"`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	expr := fs.Arg(0)
	entries, err := cron.Parse(expr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "cron: %v\n", err)
		return 1
	}
	if !*inline {
		_, _ = fmt.Fprintln(stdout, "schedule:")
	}
	for _, e := range entries {
		_, _ = fmt.Fprintln(stdout, formatEntry(e, !*inline))
	}
	return 0
}

// formatEntry returns a single-line flow-style mapping for a calendar
// entry. The indent flag adds the two-space prefix used when the
// `schedule:` header is present.
func formatEntry(e plist.CalendarEntry, indent bool) string {
	parts := make([]string, 0, 5)
	add := func(k string, p *int) {
		if p == nil {
			return
		}
		parts = append(parts, k+": "+strconv.Itoa(*p))
	}
	add("minute", e.Minute)
	add("hour", e.Hour)
	add("day", e.Day)
	add("month", e.Month)
	add("weekday", e.Weekday)
	body := "- {" + joinComma(parts) + "}"
	if indent {
		return "  " + body
	}
	return body
}

func joinComma(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += ", " + p
	}
	return out
}
