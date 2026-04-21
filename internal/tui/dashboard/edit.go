package dashboard

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/lctl/internal/editor"
	"github.com/flexphere/lctl/internal/plist"
	"github.com/flexphere/lctl/internal/yamlplist"
)

// EditFlow owns the external-editor round-trip for create/edit. It is
// injected into Model so tests can swap in a no-network fake.
type EditFlow struct {
	Store      EditStore
	Client     Registrar
	ScriptsDir string
}

// EditStore is the plist read/write surface used by the flow.
type EditStore interface {
	Load(label string) (*plist.Agent, error)
	Save(a *plist.Agent) error
	Path(label string) (string, error)
	Delete(label string) error
}

// Registrar rebinds the service after writing, so edits take effect.
type Registrar interface {
	Bootstrap(ctx context.Context, plistPath string) error
	Bootout(ctx context.Context, label string) error
}

// Prepared carries a temp-file path plus the label (empty for new
// agents) between the prepare and finalize phases.
type Prepared struct {
	TmpPath string
	Label   string
}

// PrepareEditMsg is dispatched when the prepare step completes.
type PrepareEditMsg struct {
	Prep Prepared
	Err  error
}

// FinalizeMsg is dispatched after the editor exits, reporting the
// outcome of the save pipeline.
type FinalizeMsg struct {
	Label string
	Err   error
}

// PrepareEdit loads an existing agent, converts it to YAML, and writes
// it to a temp file. It returns the temp path along with the label for
// the downstream editor step.
func (f *EditFlow) PrepareEdit(label string) tea.Cmd {
	return func() tea.Msg {
		agent, err := f.Store.Load(label)
		if err != nil {
			return PrepareEditMsg{Err: fmt.Errorf("load %s: %w", label, err)}
		}
		doc := yamlplist.FromAgent(agent)
		data, err := yamlplist.Encode(doc)
		if err != nil {
			return PrepareEditMsg{Err: fmt.Errorf("encode yaml: %w", err)}
		}
		tmp, err := writeTemp(label, data)
		if err != nil {
			return PrepareEditMsg{Err: err}
		}
		return PrepareEditMsg{Prep: Prepared{TmpPath: tmp, Label: label}}
	}
}

// PrepareNew copies a template file into a temp path so the user can
// edit it without touching the original on disk.
func (f *EditFlow) PrepareNew(templatePath string) tea.Cmd {
	return func() tea.Msg {
		data, err := os.ReadFile(templatePath) //nolint:gosec // path comes from our own template picker
		if err != nil {
			return PrepareEditMsg{Err: fmt.Errorf("read template: %w", err)}
		}
		tmp, err := writeTemp("new", data)
		if err != nil {
			return PrepareEditMsg{Err: err}
		}
		return PrepareEditMsg{Prep: Prepared{TmpPath: tmp}}
	}
}

// Finalize reads the edited YAML back, parses it, and persists the
// resulting plist. Reload via launchctl is performed only when the
// agent validated and saved cleanly.
func (f *EditFlow) Finalize(prep Prepared, editorErr error) tea.Cmd {
	return func() tea.Msg {
		defer func() { _ = os.Remove(prep.TmpPath) }()
		if editorErr != nil {
			return FinalizeMsg{Label: prep.Label, Err: fmt.Errorf("editor: %w", editorErr)}
		}
		data, err := os.ReadFile(prep.TmpPath) //nolint:gosec // temp path we created
		if err != nil {
			return FinalizeMsg{Label: prep.Label, Err: fmt.Errorf("read edited yaml: %w", err)}
		}
		doc, err := yamlplist.Decode(data)
		if err != nil {
			return FinalizeMsg{Label: prep.Label, Err: fmt.Errorf("parse yaml: %w", err)}
		}
		agent := doc.ToAgent(yamlplist.DefaultOptions(f.ScriptsDir))
		if vr := plist.Validate(agent); vr.HasErrors() {
			return FinalizeMsg{Label: prep.Label, Err: vr.Err()}
		}
		if err := f.Store.Save(agent); err != nil {
			return FinalizeMsg{Label: prep.Label, Err: fmt.Errorf("save plist: %w", err)}
		}
		ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
		defer cancel()
		// If the user renamed the label during edit, retire the old
		// plist and runtime registration so the file system doesn't
		// accumulate orphans.
		if prep.Label != "" && prep.Label != agent.Label {
			if f.Client != nil {
				_ = f.Client.Bootout(ctx, prep.Label)
			}
			_ = f.Store.Delete(prep.Label)
		}
		if f.Client != nil {
			path, err := f.Store.Path(agent.Label)
			if err != nil {
				return FinalizeMsg{Label: agent.Label, Err: err}
			}
			_ = f.Client.Bootout(ctx, agent.Label)
			if err := f.Client.Bootstrap(ctx, path); err != nil {
				return FinalizeMsg{Label: agent.Label, Err: fmt.Errorf("bootstrap: %w", err)}
			}
		}
		return FinalizeMsg{Label: agent.Label}
	}
}

// EditorCmd returns a tea.ExecProcess wrapper for the given prep. The
// editor is resolved via internal/editor.
func EditorCmd(prep Prepared) tea.Cmd {
	cmd, err := editor.Command(prep.TmpPath)
	if err != nil {
		return func() tea.Msg {
			return FinalizeMsg{Label: prep.Label, Err: err}
		}
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return finalizePendingMsg{Prep: prep, Err: err}
	})
}

// finalizePendingMsg is an internal handoff from ExecProcess back to
// the Model so it can route through EditFlow.Finalize.
type finalizePendingMsg struct {
	Prep Prepared
	Err  error
}

// writeTemp writes data to a uniquely-named yaml file under os.TempDir.
func writeTemp(label string, data []byte) (string, error) {
	base := "lctl-edit"
	if label != "" {
		base = "lctl-" + sanitizeFileName(label)
	}
	f, err := os.CreateTemp("", base+"-*.yaml")
	if err != nil {
		return "", fmt.Errorf("create temp: %w", err)
	}
	path := f.Name()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("write temp: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("close temp: %w", err)
	}
	return path, nil
}

// sanitizeFileName replaces characters that aren't safe for a file
// name, keeping label-friendly letters.
func sanitizeFileName(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '.', c == '-', c == '_':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

// LogFileFor resolves the log file the `l` key should open. It
// prefers StandardOutPath but falls back to StandardErrorPath.
func LogFileFor(agent *plist.Agent) (string, error) {
	if agent == nil {
		return "", errors.New("agent not loaded")
	}
	if agent.StandardOutPath != "" {
		return agent.StandardOutPath, nil
	}
	if agent.StandardErrorPath != "" {
		return agent.StandardErrorPath, nil
	}
	return "", errors.New("no log path configured in plist")
}

// ensureLogFileExists makes sure the editor doesn't open a brand new
// blank buffer when the log hasn't been written yet; it creates an
// empty file if missing.
func ensureLogFileExists(path string) error {
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}
