package launchd

import (
	"strings"
	"testing"
)

func TestParsePrintDisabled(t *testing.T) {
	input := `disabled services = {
		"com.apple.foo" => disabled
		"com.example.bar" => enabled
		"com.example.baz" => disabled
	}
`
	got, err := parsePrintDisabled(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if got["com.apple.foo"] != true {
		t.Errorf("foo should be disabled")
	}
	if got["com.example.bar"] != false {
		t.Errorf("bar should be enabled (false)")
	}
	if got["com.example.baz"] != true {
		t.Errorf("baz should be disabled")
	}
	if _, ok := got["never.listed"]; ok {
		t.Error("missing key should stay absent")
	}
}

func TestParsePrintDisabledIgnoresNoise(t *testing.T) {
	input := `disabled services = {
		garbage line
		"valid.label" => disabled
	}
`
	got, err := parsePrintDisabled(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got["valid.label"] {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestParsePrintDisabledEmpty(t *testing.T) {
	got, err := parsePrintDisabled(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("empty input should yield empty map")
	}
}
