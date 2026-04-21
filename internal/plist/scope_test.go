package plist

import "testing"

func TestHasLctlPrefix(t *testing.T) {
	if !HasLctlPrefix("lctl.foo") {
		t.Error("should detect prefix")
	}
	if HasLctlPrefix("foo") {
		t.Error("should not match bare label")
	}
	if HasLctlPrefix("") {
		t.Error("empty should not match")
	}
}

func TestStripLctlPrefix(t *testing.T) {
	cases := map[string]string{
		"lctl.foo":      "foo",
		"lctl.com.bar":  "com.bar",
		"foo":           "foo",
		"":              "",
		"lctl.lctl.foo": "lctl.foo",
	}
	for in, want := range cases {
		if got := StripLctlPrefix(in); got != want {
			t.Errorf("Strip(%q): got %q want %q", in, got, want)
		}
	}
}

func TestAddLctlPrefixIdempotent(t *testing.T) {
	cases := map[string]string{
		"foo":          "lctl.foo",
		"lctl.foo":     "lctl.foo",
		"com.bar":      "lctl.com.bar",
		"lctl.com.bar": "lctl.com.bar",
		"":             "",
	}
	for in, want := range cases {
		if got := AddLctlPrefix(in); got != want {
			t.Errorf("Add(%q): got %q want %q", in, got, want)
		}
	}
}
