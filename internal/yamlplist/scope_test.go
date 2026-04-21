package yamlplist

import (
	"testing"

	"github.com/flexphere/lctl/internal/plist"
)

func TestToAgentAddsLctlPrefix(t *testing.T) {
	d := &Document{Label: "com.flexphere.foo", Program: "/bin/true"}
	a := d.ToAgent(opts())
	if a.Label != "lctl.com.flexphere.foo" {
		t.Errorf("prefix not added: %q", a.Label)
	}
}

func TestToAgentIdempotentForExistingPrefix(t *testing.T) {
	d := &Document{Label: "lctl.com.flexphere.foo", Program: "/bin/true"}
	a := d.ToAgent(opts())
	if a.Label != "lctl.com.flexphere.foo" {
		t.Errorf("prefix should not be doubled: %q", a.Label)
	}
}

func TestToAgentEmptyLabelStaysEmpty(t *testing.T) {
	d := &Document{Label: "", Program: "/bin/true"}
	a := d.ToAgent(opts())
	if a.Label != "" {
		t.Errorf("empty label should stay empty so validation flags it: %q", a.Label)
	}
}

func TestFromAgentStripsPrefix(t *testing.T) {
	a := &plist.Agent{Label: "lctl.com.flexphere.foo"}
	d := FromAgent(a)
	if d.Label != "com.flexphere.foo" {
		t.Errorf("prefix not stripped: %q", d.Label)
	}
}

func TestFromAgentLeavesUnprefixedAlone(t *testing.T) {
	a := &plist.Agent{Label: "com.apple.trustd.agent"}
	d := FromAgent(a)
	if d.Label != "com.apple.trustd.agent" {
		t.Errorf("non-prefixed label altered: %q", d.Label)
	}
}

func TestRoundtripBareLabelStable(t *testing.T) {
	// User writes bare label → save adds prefix → edit strips it back
	// → save re-adds the same prefix. The user's view never changes.
	doc1 := &Document{Label: "com.flexphere.bk", Program: "/bin/true"}
	agent := doc1.ToAgent(opts())
	doc2 := FromAgent(agent)
	if doc2.Label != doc1.Label {
		t.Errorf("roundtrip drift: %q → %q", doc1.Label, doc2.Label)
	}
}
