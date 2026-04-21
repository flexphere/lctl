package launchd

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// disabledLinePattern matches a single entry inside the body of
// `launchctl print-disabled <domain>`, e.g.
//
//	"com.apple.Siri.agent" => enabled
//	"com.example.foo" => disabled
//
// Surrounding whitespace and tabs are tolerated.
var disabledLinePattern = regexp.MustCompile(`^\s*"([^"]+)"\s*=>\s*(enabled|disabled)\s*$`)

// parsePrintDisabled converts the output of `launchctl print-disabled`
// into a label→disabled map. Only labels explicitly marked `disabled`
// are set to true; anything else (including `enabled`) defaults to
// false so the caller can treat missing keys as enabled too.
func parsePrintDisabled(r io.Reader) (map[string]bool, error) {
	out := map[string]bool{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		m := disabledLinePattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		out[m[1]] = m[2] == "disabled"
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan print-disabled: %w", err)
	}
	return out, nil
}

// parseList parses the tab-separated output of `launchctl list`.
// Expected format:
//
//	PID\tStatus\tLabel
//	638\t0\tcom.apple.example
//	-\t0\tcom.apple.idle
func parseList(r io.Reader) ([]ListEntry, error) {
	var out []ListEntry
	sc := bufio.NewScanner(r)
	first := true
	for sc.Scan() {
		line := sc.Text()
		if first {
			first = false
			if strings.HasPrefix(line, "PID\t") || strings.HasPrefix(line, "PID ") {
				continue
			}
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		entry, err := parseListLine(line)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan list: %w", err)
	}
	return out, nil
}

func parseListLine(line string) (ListEntry, error) {
	fields := strings.SplitN(line, "\t", 3)
	if len(fields) != 3 {
		return ListEntry{}, fmt.Errorf("unexpected list line %q", line)
	}
	pidStr := strings.TrimSpace(fields[0])
	statusStr := strings.TrimSpace(fields[1])
	label := strings.TrimSpace(fields[2])
	if label == "" {
		return ListEntry{}, fmt.Errorf("empty label in line %q", line)
	}
	entry := ListEntry{Label: label}
	if pidStr != "-" {
		n, err := strconv.Atoi(pidStr)
		if err != nil {
			return ListEntry{}, fmt.Errorf("parse pid %q: %w", pidStr, err)
		}
		entry.PID = &n
	}
	if statusStr != "-" {
		n, err := strconv.Atoi(statusStr)
		if err != nil {
			return ListEntry{}, fmt.Errorf("parse status %q: %w", statusStr, err)
		}
		entry.LastExit = &n
	}
	return entry, nil
}
