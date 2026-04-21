package plist

import "strings"

// LctlPrefix is the label namespace lctl uses to scope the agents it
// manages. Any agent created or edited through lctl receives this
// prefix before being written to disk, and the Dashboard lists only
// agents whose labels start with it. The trailing dot keeps reverse-DNS
// labels well-formed (e.g. "lctl.com.flexphere.backup").
const LctlPrefix = "lctl."

// HasLctlPrefix reports whether label is scoped to lctl.
func HasLctlPrefix(label string) bool { return strings.HasPrefix(label, LctlPrefix) }

// StripLctlPrefix removes one leading "lctl." namespace from label.
// Labels without the prefix pass through unchanged.
func StripLctlPrefix(label string) string { return strings.TrimPrefix(label, LctlPrefix) }

// AddLctlPrefix ensures label carries exactly one "lctl." prefix. An
// empty label returns empty so validation can report the missing field
// rather than letting a bare "lctl." slip through.
func AddLctlPrefix(label string) string {
	if label == "" {
		return ""
	}
	return LctlPrefix + strings.TrimPrefix(label, LctlPrefix)
}
