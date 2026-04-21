package plist

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return &Store{Dir: t.TempDir()}
}

func TestStoreSaveLoadRoundTrip(t *testing.T) {
	s := newTestStore(t)
	a := &Agent{
		Label:         "com.flexphere.save",
		Program:       "/bin/true",
		RunAtLoad:     true,
		StartInterval: 60,
	}
	if err := s.Save(a); err != nil {
		t.Fatalf("save: %v", err)
	}
	p, _ := s.Path(a.Label)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("file not written: %v", err)
	}

	loaded, err := s.Load(a.Label)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Label != a.Label || loaded.Program != a.Program || loaded.StartInterval != 60 {
		t.Errorf("mismatch: %+v", loaded)
	}
}

func TestStoreListIgnoresNonPlist(t *testing.T) {
	s := newTestStore(t)
	a := &Agent{Label: "com.flexphere.one", Program: "/bin/true"}
	if err := s.Save(a); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir, "README"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	agents, errs := s.List()
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(agents) != 1 {
		t.Fatalf("want 1 agent, got %d", len(agents))
	}
	if agents[0].Label != a.Label {
		t.Errorf("label mismatch: %q", agents[0].Label)
	}
}

func TestStoreListBadFilesAreCollected(t *testing.T) {
	s := newTestStore(t)
	if err := os.WriteFile(filepath.Join(s.Dir, "broken.plist"), []byte("not xml"), 0o644); err != nil {
		t.Fatal(err)
	}
	agents, errs := s.List()
	if len(errs) == 0 {
		t.Error("expected error for broken plist")
	}
	if len(agents) != 0 {
		t.Errorf("want 0 agents, got %d", len(agents))
	}
}

func TestStorePathRejectsTraversal(t *testing.T) {
	s := newTestStore(t)
	bad := []string{"../evil", "a/b", "", "dir\\name", "../../root"}
	for _, label := range bad {
		if _, err := s.Path(label); err == nil {
			t.Errorf("expected error for label %q", label)
		}
	}
}

func TestStoreSaveAtomic(t *testing.T) {
	s := newTestStore(t)
	a := &Agent{Label: "com.flexphere.atomic", Program: "/bin/true"}
	if err := s.Save(a); err != nil {
		t.Fatal(err)
	}
	a2 := &Agent{Label: "com.flexphere.atomic", Program: "/bin/false"}
	if err := s.Save(a2); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load(a.Label)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Program != "/bin/false" {
		t.Errorf("expected overwrite, got %q", loaded.Program)
	}
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("tmp file leaked: %s", e.Name())
		}
	}
}

func TestStoreDeleteMissingOK(t *testing.T) {
	s := newTestStore(t)
	if err := s.Delete("com.flexphere.nope"); err != nil {
		t.Errorf("delete missing should be ok: %v", err)
	}
}

func TestStoreDeleteExisting(t *testing.T) {
	s := newTestStore(t)
	a := &Agent{Label: "com.flexphere.del", Program: "/bin/true"}
	if err := s.Save(a); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(a.Label); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Load(a.Label); err == nil {
		t.Error("expected load error after delete")
	}
}
