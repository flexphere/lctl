package dashboard

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flexphere/lctl/internal/plist"
)

type fakeEditStore struct {
	agent   *plist.Agent
	loadErr error
	saved   *plist.Agent
	saveErr error
	pathErr error
	deleted []string
}

func (f *fakeEditStore) Load(string) (*plist.Agent, error) { return f.agent, f.loadErr }
func (f *fakeEditStore) Save(a *plist.Agent) error         { f.saved = a; return f.saveErr }
func (f *fakeEditStore) Path(l string) (string, error) {
	if f.pathErr != nil {
		return "", f.pathErr
	}
	return "/tmp/" + l + ".plist", nil
}
func (f *fakeEditStore) Delete(label string) error {
	f.deleted = append(f.deleted, label)
	return nil
}

type fakeReg struct {
	bootoutLabels  []string
	bootstrapPaths []string
	bootstrapErr   error
}

func (f *fakeReg) Bootstrap(_ context.Context, path string) error {
	f.bootstrapPaths = append(f.bootstrapPaths, path)
	return f.bootstrapErr
}
func (f *fakeReg) Bootout(_ context.Context, label string) error {
	f.bootoutLabels = append(f.bootoutLabels, label)
	return nil
}

func writeYAMLTemp(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "edit.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPrepareEditWritesTempYAML(t *testing.T) {
	store := &fakeEditStore{agent: &plist.Agent{Label: "com.x", Program: "/bin/true"}}
	flow := &EditFlow{Store: store}
	cmd := flow.PrepareEdit("com.x")
	msg := cmd().(PrepareEditMsg)
	if msg.Err != nil {
		t.Fatalf("prepare failed: %v", msg.Err)
	}
	data, err := os.ReadFile(msg.Prep.TmpPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "com.x") {
		t.Errorf("temp yaml missing label: %q", string(data))
	}
	_ = os.Remove(msg.Prep.TmpPath)
}

func TestPrepareEditLoadError(t *testing.T) {
	store := &fakeEditStore{loadErr: errors.New("missing")}
	flow := &EditFlow{Store: store}
	msg := flow.PrepareEdit("com.x")().(PrepareEditMsg)
	if msg.Err == nil {
		t.Error("expected error")
	}
}

func TestPrepareNewCopiesTemplate(t *testing.T) {
	src := writeYAMLTemp(t, "label: com.x\nprogram: /bin/true\n")
	flow := &EditFlow{}
	msg := flow.PrepareNew(src)().(PrepareEditMsg)
	if msg.Err != nil {
		t.Fatal(msg.Err)
	}
	data, _ := os.ReadFile(msg.Prep.TmpPath)
	if !strings.Contains(string(data), "com.x") {
		t.Errorf("template not copied: %s", string(data))
	}
	_ = os.Remove(msg.Prep.TmpPath)
}

func TestPrepareNewMissingTemplate(t *testing.T) {
	flow := &EditFlow{}
	msg := flow.PrepareNew("/does/not/exist")().(PrepareEditMsg)
	if msg.Err == nil {
		t.Error("expected error")
	}
}

func TestFinalizeSavesAndBootstraps(t *testing.T) {
	tmp := writeYAMLTemp(t, `label: com.flexphere.ok
program: /bin/true
run_at_load: true
`)
	store := &fakeEditStore{}
	reg := &fakeReg{}
	flow := &EditFlow{Store: store, Client: reg}
	msg := flow.Finalize(Prepared{TmpPath: tmp, Label: "com.flexphere.ok"}, nil)().(FinalizeMsg)
	if msg.Err != nil {
		t.Fatalf("finalize: %v", msg.Err)
	}
	if store.saved == nil || store.saved.Label != "lctl.com.flexphere.ok" {
		t.Errorf("save not called or prefix not added: %+v", store.saved)
	}
	if len(reg.bootstrapPaths) == 0 {
		t.Error("expected bootstrap call")
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("temp file should be cleaned up")
	}
}

func TestFinalizeEditorErrorPropagates(t *testing.T) {
	tmp := writeYAMLTemp(t, "label: com.x\n")
	flow := &EditFlow{Store: &fakeEditStore{}}
	msg := flow.Finalize(Prepared{TmpPath: tmp, Label: "com.x"}, errors.New("killed"))().(FinalizeMsg)
	if msg.Err == nil || !strings.Contains(msg.Err.Error(), "editor") {
		t.Errorf("expected editor error, got %v", msg.Err)
	}
}

func TestFinalizeInvalidYAML(t *testing.T) {
	tmp := writeYAMLTemp(t, "{{{ not yaml")
	flow := &EditFlow{Store: &fakeEditStore{}}
	msg := flow.Finalize(Prepared{TmpPath: tmp, Label: "com.x"}, nil)().(FinalizeMsg)
	if msg.Err == nil {
		t.Error("expected parse error")
	}
}

func TestFinalizeValidationFailure(t *testing.T) {
	tmp := writeYAMLTemp(t, `label: ""
program: /bin/true
`)
	flow := &EditFlow{Store: &fakeEditStore{}}
	msg := flow.Finalize(Prepared{TmpPath: tmp}, nil)().(FinalizeMsg)
	if msg.Err == nil {
		t.Error("expected validation error")
	}
}

func TestFinalizeRenameRetiresOldLabel(t *testing.T) {
	tmp := writeYAMLTemp(t, `label: com.flexphere.renamed
program: /bin/true
run_at_load: true
`)
	store := &fakeEditStore{}
	reg := &fakeReg{}
	flow := &EditFlow{Store: store, Client: reg}
	msg := flow.Finalize(Prepared{TmpPath: tmp, Label: "com.flexphere.original"}, nil)().(FinalizeMsg)
	if msg.Err != nil {
		t.Fatal(msg.Err)
	}
	found := false
	for _, d := range store.deleted {
		if d == "com.flexphere.original" {
			found = true
		}
	}
	if !found {
		t.Errorf("old label not deleted from store: %v", store.deleted)
	}
	found = false
	for _, b := range reg.bootoutLabels {
		if b == "com.flexphere.original" {
			found = true
		}
	}
	if !found {
		t.Errorf("old label not booted out: %v", reg.bootoutLabels)
	}
}

func TestFinalizeSaveError(t *testing.T) {
	tmp := writeYAMLTemp(t, `label: com.flexphere.ok
program: /bin/true
run_at_load: true
`)
	store := &fakeEditStore{saveErr: errors.New("disk")}
	flow := &EditFlow{Store: store}
	msg := flow.Finalize(Prepared{TmpPath: tmp}, nil)().(FinalizeMsg)
	if msg.Err == nil || !strings.Contains(msg.Err.Error(), "save plist") {
		t.Errorf("expected save error: %v", msg.Err)
	}
}

func TestLogFileForPrefersStdout(t *testing.T) {
	a := &plist.Agent{StandardOutPath: "/tmp/out", StandardErrorPath: "/tmp/err"}
	p, err := LogFileFor(a)
	if err != nil || p != "/tmp/out" {
		t.Errorf("expected stdout, got %q / %v", p, err)
	}
}

func TestLogFileForFallsBackToStderr(t *testing.T) {
	a := &plist.Agent{StandardErrorPath: "/tmp/err"}
	p, err := LogFileFor(a)
	if err != nil || p != "/tmp/err" {
		t.Errorf("expected err, got %q / %v", p, err)
	}
}

func TestLogFileForNoConfig(t *testing.T) {
	_, err := LogFileFor(&plist.Agent{})
	if err == nil {
		t.Error("expected error")
	}
}

func TestEnsureLogFileExistsCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "log.txt")
	if err := ensureLogFileExists(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}

func TestEnsureLogFileExistsPreservesContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureLogFileExists(path); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "hello" {
		t.Errorf("existing content overwritten")
	}
}

func TestSanitizeFileName(t *testing.T) {
	if got := sanitizeFileName("com.flexphere/../../etc"); strings.Contains(got, "/") {
		t.Errorf("unsafe char leaked: %q", got)
	}
	if got := sanitizeFileName("a.b-c_1"); got != "a.b-c_1" {
		t.Errorf("safe chars dropped: %q", got)
	}
}
