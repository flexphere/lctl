package plist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRequiresLabelAndProgram(t *testing.T) {
	r := Validate(&Agent{})
	if !r.HasErrors() {
		t.Fatal("expected errors for empty agent")
	}
	var haveLabel, haveProgram bool
	for _, i := range r.Issues {
		if !i.Warning && i.Field == "Label" {
			haveLabel = true
		}
		if !i.Warning && i.Field == "Program" {
			haveProgram = true
		}
	}
	if !haveLabel || !haveProgram {
		t.Errorf("missing expected errors: %+v", r.Issues)
	}
}

func TestValidateLabelFormat(t *testing.T) {
	r := Validate(&Agent{Label: "flat", Program: "/bin/true"})
	found := false
	for _, i := range r.Issues {
		if i.Field == "Label" && !i.Warning && strings.Contains(i.Message, "reverse-DNS") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected reverse-DNS complaint: %+v", r.Issues)
	}
}

func TestValidateExecutableOK(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := Validate(&Agent{Label: "com.flexphere.ok", Program: script, RunAtLoad: true})
	if r.HasErrors() {
		t.Errorf("unexpected errors: %+v", r.Issues)
	}
}

func TestValidateNotExecutable(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := Validate(&Agent{Label: "com.flexphere.ok", Program: script, RunAtLoad: true})
	found := false
	for _, i := range r.Issues {
		if i.Field == "Program" && i.Warning && strings.Contains(i.Message, "not executable") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected executable warning: %+v", r.Issues)
	}
}

func TestValidateRelativePaths(t *testing.T) {
	r := Validate(&Agent{
		Label:            "com.flexphere.rel",
		Program:          "relative/path",
		WorkingDirectory: "wd",
	})
	var haveExec, haveWD bool
	for _, i := range r.Issues {
		if !i.Warning && i.Field == "Program" && strings.Contains(i.Message, "absolute") {
			haveExec = true
		}
		if !i.Warning && i.Field == "WorkingDirectory" && strings.Contains(i.Message, "absolute") {
			haveWD = true
		}
	}
	if !haveExec || !haveWD {
		t.Errorf("expected absolute-path errors: %+v", r.Issues)
	}
}

func TestValidateNoTriggerIsWarning(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := Validate(&Agent{Label: "com.flexphere.notrig", Program: script})
	if r.HasErrors() {
		t.Fatalf("should be warning only: %+v", r.Issues)
	}
	found := false
	for _, i := range r.Issues {
		if i.Warning && i.Field == "Schedule" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected schedule warning: %+v", r.Issues)
	}
}

func TestValidateResultErr(t *testing.T) {
	r := ValidationResult{Issues: []ValidationIssue{
		{Field: "Label", Message: "x", Warning: false},
		{Field: "other", Message: "y", Warning: true},
	}}
	if r.Err() == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(r.Err().Error(), "Label") {
		t.Errorf("error missing field: %v", r.Err())
	}
	if strings.Contains(r.Err().Error(), "other") {
		t.Errorf("warning should not be in err: %v", r.Err())
	}
}
