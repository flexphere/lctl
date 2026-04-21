package plist

import (
	"strings"
	"testing"
)

func TestKeepAliveActiveBool(t *testing.T) {
	if KeepAliveActive(nil) {
		t.Error("nil should be inactive")
	}
	if KeepAliveActive(false) {
		t.Error("false should be inactive")
	}
	if !KeepAliveActive(true) {
		t.Error("true should be active")
	}
}

func TestKeepAliveActiveDict(t *testing.T) {
	dict := map[string]any{"SuccessfulExit": false}
	if !KeepAliveActive(dict) {
		t.Error("dict form should be considered active")
	}
	// An empty dict still evaluates to active — launchd handles it.
	if !KeepAliveActive(map[string]any{}) {
		t.Error("empty dict is still dict form")
	}
}

func TestAgentKindRespectsDictKeepAlive(t *testing.T) {
	a := &Agent{Label: "x", KeepAlive: map[string]any{"Crashed": true}}
	if a.Kind() != ScheduleDaemon {
		t.Errorf("dict keep_alive should be daemon, got %v", a.Kind())
	}
}

func TestEncodeKeepAliveBool(t *testing.T) {
	a := &Agent{Label: "com.x", Program: "/bin/true", KeepAlive: true}
	data, err := EncodeBytes(a)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "<key>KeepAlive</key>") {
		t.Errorf("missing KeepAlive key:\n%s", s)
	}
	if !strings.Contains(s, "<true/>") {
		t.Errorf("expected <true/>: %s", s)
	}
}

func TestEncodeKeepAliveDict(t *testing.T) {
	a := &Agent{
		Label:     "com.x",
		Program:   "/bin/true",
		KeepAlive: map[string]any{"SuccessfulExit": false, "Crashed": true},
	}
	data, err := EncodeBytes(a)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "<key>KeepAlive</key>") {
		t.Errorf("missing KeepAlive key:\n%s", s)
	}
	// The dict form emits a nested <dict>.
	if !strings.Contains(s, "<key>SuccessfulExit</key>") {
		t.Errorf("missing SuccessfulExit subkey:\n%s", s)
	}
	if !strings.Contains(s, "<key>Crashed</key>") {
		t.Errorf("missing Crashed subkey:\n%s", s)
	}
}

func TestDecodeKeepAliveDict(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.x</string>
  <key>Program</key><string>/bin/true</string>
  <key>KeepAlive</key>
  <dict>
    <key>SuccessfulExit</key><false/>
  </dict>
</dict>
</plist>`
	a, err := DecodeBytes([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	if a.KeepAlive == nil {
		t.Fatal("KeepAlive lost on decode")
	}
	if !KeepAliveActive(a.KeepAlive) {
		t.Error("dict KeepAlive should be active")
	}
}
