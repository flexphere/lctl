package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/flexphere/lctl/internal/config"
	"github.com/flexphere/lctl/internal/editor"
	"github.com/flexphere/lctl/internal/launchd"
	"github.com/flexphere/lctl/internal/plist"
	"github.com/flexphere/lctl/internal/service"
	"github.com/flexphere/lctl/internal/tui/common"
)

// loadTimeout bounds launchctl calls that back dashboard refreshes.
const loadTimeout = 5 * time.Second

// opTimeout bounds per-job operations.
const opTimeout = 10 * time.Second

// JobsLoadedMsg delivers a fresh list of jobs to the Dashboard.
type JobsLoadedMsg struct {
	Result service.ListResult
	Err    error
}

// OpCompletedMsg reports the result of an operation triggered by a key.
type OpCompletedMsg struct {
	Op    string
	Label string
	Err   error
}

// autoRefreshMsg is the tick that drives periodic reloads.
type autoRefreshMsg struct{}

// Loader describes what the Dashboard needs to populate the list.
type Loader interface {
	List(ctx context.Context) (service.ListResult, error)
}

// Settings groups the user-configurable runtime knobs the Dashboard
// cares about. Passing a zero value yields sensible defaults.
type Settings struct {
	Keymap          config.Keymap
	AutoRefresh     bool
	RefreshInterval time.Duration
	Layout          config.LayoutSettings
	Variables       map[string]config.VariableSettings
	Filter          config.FilterSettings
}

// Model is the Dashboard screen.
type Model struct {
	list     list.Model
	loader   Loader
	ops      service.Ops
	flow     *EditFlow
	renderer *rowRenderer
	filter   config.FilterSettings
	keys     config.Keymap
	auto     bool
	interval time.Duration
	jobs     []service.Job
	issues   []error
	loadErr  error
	opStatus string
	loaded   bool
	width    int
	height   int
}

// New builds a Dashboard model backed by bubbles/list, with the fzf-
// inspired compact delegate and built-in incremental filter.
func New(loader Loader, ops service.Ops, flow *EditFlow, settings Settings) Model {
	keys := settings.Keymap
	defaults := config.DefaultSettings()
	if keys.Edit == "" {
		keys = defaults.Keymap
	}
	interval := settings.RefreshInterval
	if interval <= 0 {
		interval = time.Duration(defaults.List.RefreshInterval) * time.Second
	}
	layout := settings.Layout
	if layout.Template == "" {
		layout.Template = defaults.List.Layout.Template
	}
	if layout.Divider == "" {
		layout.Divider = defaults.List.Layout.Divider
	}
	variables := settings.Variables
	if variables == nil {
		variables = defaults.List.Variables
	}
	filter := settings.Filter
	if filter.Prompt == "" {
		filter.Prompt = defaults.Filter.Prompt
	}
	renderer := newRowRenderer(layout, variables)
	l := list.New(nil, delegate{renderer: renderer}, 0, 0)
	// Title, status (item count + filter), column header, and the
	// filter prompt are all rendered by Model.View above the list so
	// they stack in the order the user expects. `SetShowFilter(false)`
	// suppresses list's built-in filter bar while leaving the filter
	// state machine enabled — we reach into list.FilterInput.View()
	// ourselves to render the prompt in itemStatusLine.
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowPagination(true)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings() // `q` is handled by our key layer
	l.FilterInput.Prompt = filter.Prompt
	l.FilterInput.PromptStyle = styleForColor(filter.PromptColor)
	l.FilterInput.TextStyle = styleForColor(filter.TextColor)
	l.FilterInput.Cursor.Style = styleForColor(filter.TextColor)
	return Model{
		list:     l,
		loader:   loader,
		ops:      ops,
		flow:     flow,
		renderer: renderer,
		filter:   filter,
		keys:     keys,
		auto:     settings.AutoRefresh,
		interval: interval,
	}
}

// styleForColor returns a lipgloss style with the given foreground
// color, or an unstyled style when the color string is empty. Used to
// apply optional color overrides to bubbles/list's FilterInput without
// hard-coding a default palette.
func styleForColor(c string) lipgloss.Style {
	if c == "" {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c))
}

// Init triggers the first load and, if auto-refresh is enabled, the
// first periodic tick.
func (m Model) Init() tea.Cmd {
	if m.auto {
		return tea.Batch(m.loadCmd(), m.tickCmd())
	}
	return m.loadCmd()
}

// Update advances model state.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve two rows for our manual title + header plus four for
		// the footer (load status, op status, help line, margin).
		m.list.SetSize(msg.Width, max(msg.Height-6, 3))
		// Let the renderer stretch its label column into any surplus
		// the terminal gives us so the list fills the row width.
		if m.renderer != nil {
			m.renderer.SetTerminalWidth(msg.Width)
		}
	case tea.KeyMsg:
		return m.handleKey(msg)
	case JobsLoadedMsg:
		m.jobs = msg.Result.Jobs
		m.issues = msg.Result.PlistIssues
		m.loadErr = msg.Err
		m.loaded = true
		m.list.SetItems(toItems(msg.Result.Jobs))
	case OpCompletedMsg:
		if msg.Err != nil {
			m.opStatus = fmt.Sprintf("%s %s: %v", msg.Op, msg.Label, msg.Err)
		} else {
			m.opStatus = fmt.Sprintf("%s %s: ok", msg.Op, msg.Label)
		}
		return m, m.loadCmd()
	case PrepareEditMsg:
		if msg.Err != nil {
			m.opStatus = "edit: " + msg.Err.Error()
			return m, nil
		}
		return m, EditorCmd(msg.Prep)
	case finalizePendingMsg:
		if m.flow == nil {
			return m, nil
		}
		return m, m.flow.Finalize(msg.Prep, msg.Err)
	case FinalizeMsg:
		if msg.Err != nil {
			m.opStatus = "edit " + msg.Label + ": " + msg.Err.Error()
		} else {
			m.opStatus = "edit " + msg.Label + ": ok"
		}
		return m, m.loadCmd()
	case autoRefreshMsg:
		if !m.auto {
			return m, nil
		}
		return m, tea.Batch(m.loadCmd(), m.tickCmd())
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// SelectedLabel returns the fully-qualified (lctl-prefixed) label
// under the cursor, or "" when the list is empty.
func (m Model) SelectedLabel() string {
	it, ok := m.list.SelectedItem().(jobItem)
	if !ok {
		return ""
	}
	return it.label
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	// When the filter prompt is active, every keystroke belongs to the
	// list's input box — otherwise our shortcuts would intercept the
	// user's search characters.
	if m.list.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
	key := msg.String()
	// Ctrl+C always quits so a keymap typo can never trap the user.
	if key == "ctrl+c" || key == m.keys.Quit {
		return m, tea.Quit
	}
	switch key {
	case m.keys.Refresh:
		m.opStatus = ""
		return m, m.loadCmd()
	case m.keys.New:
		return m, func() tea.Msg {
			return common.SwitchScreenMsg{Target: common.ScreenTemplatePicker}
		}
	case m.keys.Edit:
		return m.startEdit()
	case m.keys.Log:
		return m.openLog()
	case m.keys.Start:
		return m.runOp("start", startOp)
	case m.keys.Stop:
		return m.runOp("stop", stopOp)
	case m.keys.Restart:
		return m.runOp("restart", restartOp)
	case m.keys.Toggle:
		return m.toggleEnabled()
	case m.keys.Delete:
		return m.runOp("delete", deleteOp)
	}
	// Everything else (arrow keys, /, PgUp etc.) is list's responsibility.
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

type opFn func(ctx context.Context, ops service.Ops, label string) error

func startOp(ctx context.Context, o service.Ops, l string) error   { return o.Start(ctx, l) }
func stopOp(ctx context.Context, o service.Ops, l string) error    { return o.Stop(ctx, l) }
func restartOp(ctx context.Context, o service.Ops, l string) error { return o.Restart(ctx, l) }
func enableOp(ctx context.Context, o service.Ops, l string) error  { return o.Enable(ctx, l) }
func disableOp(ctx context.Context, o service.Ops, l string) error { return o.Disable(ctx, l) }
func deleteOp(ctx context.Context, o service.Ops, l string) error  { return o.Delete(ctx, l) }

func (m Model) runOp(name string, fn opFn) (Model, tea.Cmd) {
	label := m.SelectedLabel()
	if label == "" {
		return m, nil
	}
	if m.ops == nil {
		m.opStatus = name + ": ops not configured"
		return m, nil
	}
	m.opStatus = fmt.Sprintf("%s %s ...", name, plist.StripLctlPrefix(label))
	ops := m.ops
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
		defer cancel()
		return OpCompletedMsg{Op: name, Label: label, Err: fn(ctx, ops, label)}
	}
}

// toggleEnabled flips the enable/disable state of the selected job.
// Direction is decided from the current service.Job.Disabled field so
// a single key can replace the separate enable/disable shortcuts.
func (m Model) toggleEnabled() (Model, tea.Cmd) {
	label := m.SelectedLabel()
	if label == "" {
		return m, nil
	}
	if m.ops == nil {
		m.opStatus = "toggle: ops not configured"
		return m, nil
	}
	var job *service.Job
	for i := range m.jobs {
		if m.jobs[i].Label == label {
			job = &m.jobs[i]
			break
		}
	}
	if job == nil {
		m.opStatus = "toggle: job not loaded"
		return m, nil
	}
	if job.Disabled {
		return m.runOp("enable", enableOp)
	}
	return m.runOp("disable", disableOp)
}

// startEdit kicks off the PrepareEdit → editor → Finalize chain.
func (m Model) startEdit() (Model, tea.Cmd) {
	label := m.SelectedLabel()
	if label == "" {
		return m, nil
	}
	if m.flow == nil {
		m.opStatus = "edit: flow not configured"
		return m, nil
	}
	m.opStatus = "preparing edit..."
	return m, m.flow.PrepareEdit(label)
}

// openLog launches the editor on the selected job's stdout log.
func (m Model) openLog() (Model, tea.Cmd) {
	label := m.SelectedLabel()
	if label == "" {
		return m, nil
	}
	if m.flow == nil {
		m.opStatus = "log: flow not configured"
		return m, nil
	}
	var job *service.Job
	for i := range m.jobs {
		if m.jobs[i].Label == label {
			job = &m.jobs[i]
			break
		}
	}
	if job == nil || job.Agent == nil {
		m.opStatus = "log: agent not loaded"
		return m, nil
	}
	path, err := LogFileFor(job.Agent)
	if err != nil {
		m.opStatus = "log: " + err.Error()
		return m, nil
	}
	if err := ensureLogFileExists(path); err != nil {
		m.opStatus = "log: " + err.Error()
		return m, nil
	}
	cmd, err := editor.Command(path)
	if err != nil {
		m.opStatus = "log: " + err.Error()
		return m, nil
	}
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return OpCompletedMsg{Op: "log", Label: label, Err: err}
	})
}

// StartNew is called by the app when the user picked a template.
func (m Model) StartNew(templatePath string) (Model, tea.Cmd) {
	if m.flow == nil {
		m.opStatus = "new: flow not configured"
		return m, nil
	}
	m.opStatus = "preparing new..."
	return m, m.flow.PrepareNew(templatePath)
}

// View renders the Dashboard in the order the user reads top-to-bottom:
// a status-aggregate title (total + per-state counts), the optional
// filter prompt row, the column header row, the list body, and a
// footer with load/op status plus a keymap-aware help line.
func (m Model) View() string {
	header := common.HeaderText.Render(m.renderer.renderHeader())

	var footer []string
	switch {
	case m.loadErr != nil:
		footer = append(footer, common.StatusErrored.Render(fmt.Sprintf("load error: %v", m.loadErr)))
	case !m.loaded:
		footer = append(footer, common.HelpStyle.Render("loading..."))
	case len(m.jobs) == 0:
		footer = append(footer, common.HelpStyle.Render("no agents found under the lctl. namespace"))
	default:
		// The total count moved to aggregateLine above; keep only
		// footer-worthy auxiliary state here.
		var bits []string
		if n := len(m.issues); n > 0 {
			bits = append(bits, fmt.Sprintf("%d plist issue(s)", n))
		}
		if m.auto {
			bits = append(bits, fmt.Sprintf("auto-refresh every %s", m.interval))
		}
		if len(bits) > 0 {
			footer = append(footer, common.HelpStyle.Render(strings.Join(bits, "  |  ")))
		}
	}
	if m.opStatus != "" {
		footer = append(footer, common.HelpStyle.Render(m.opStatus))
	}
	footer = append(footer, common.HelpStyle.Render(m.helpLine()))
	parts := make([]string, 0, 7)
	if title := m.aggregateLine(); title != "" {
		// Build a title row with "lctl" bold on the left and the
		// state aggregate pushed to the right. lipgloss.Width is
		// ANSI-aware so per-segment color escapes in either half do
		// not throw off the gap calculation.
		brand := common.TitleStyle.Padding(0, 0).Render("lctl")
		aggregate := common.HelpStyle.Render(title)
		line := brand + "  " + aggregate
		if m.width > 0 {
			gap := m.width - lipgloss.Width(brand) - lipgloss.Width(aggregate)
			if gap < 1 {
				gap = 1
			}
			line = brand + strings.Repeat(" ", gap) + aggregate
		}
		// Blank row below the aggregate gives the list a bit of
		// breathing room; consistent whether or not a filter is active.
		parts = append(parts, line, "")
	}
	if status := m.itemStatusLine(); status != "" {
		parts = append(parts, common.HelpStyle.Render(status))
	}
	parts = append(parts, header, m.list.View(), lipgloss.JoinVertical(lipgloss.Left, footer...))
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// aggregateLine renders the status-summary title shown above the
// filter prompt and header. It reports the total count plus a per-
// state breakdown using the compiled-in status palette so `running`
// and `errored` pop against the neutral surrounding text. Skipped
// while loading, on error, or when no lctl-scoped agents exist —
// the footer surfaces those states instead.
func (m Model) aggregateLine() string {
	if !m.loaded || m.loadErr != nil || len(m.jobs) == 0 {
		return ""
	}
	var running, loaded, errored int
	for _, j := range m.jobs {
		switch j.State {
		case launchd.StateRunning:
			running++
		case launchd.StateLoaded:
			loaded++
		case launchd.StateErrored:
			errored++
		}
	}
	total := len(m.jobs)
	unit := "agents"
	if total == 1 {
		unit = "agent"
	}
	parts := []string{
		fmt.Sprintf("%d %s", total, unit),
		common.StatusRunning.Render(fmt.Sprintf("%d running", running)),
		common.StatusIdle.Render(fmt.Sprintf("%d loaded", loaded)),
		erroredStyle(errored).Render(fmt.Sprintf("%d errored", errored)),
	}
	return strings.Join(parts, " · ")
}

// erroredStyle returns the red status style when there is at least
// one errored job, so the count draws the eye. A zero count stays
// dim like the rest of the breakdown so it does not false-alarm.
func erroredStyle(count int) lipgloss.Style {
	if count > 0 {
		return common.StatusErrored
	}
	return common.StatusIdle
}

// itemStatusLine returns the filter prompt row to place above the
// column header. It emits an empty string in the idle state so the
// dashboard shows no redundant count; the footer still reports the
// total number of agents. While the user is still typing we render
// the list's own FilterInput (preserving cursor + edit keybindings);
// after Enter we re-render the prompt with the configured styles so
// the line keeps the same look in both states.
func (m Model) itemStatusLine() string {
	switch m.list.FilterState() {
	case list.Filtering:
		return m.list.FilterInput.View()
	case list.FilterApplied:
		promptStyle := styleForColor(m.filter.PromptColor)
		textStyle := styleForColor(m.filter.TextColor)
		return promptStyle.Render(m.filter.Prompt) + textStyle.Render(m.list.FilterValue())
	}
	return ""
}

// helpLine assembles a help bar from current key bindings so
// customizations stay visible.
func (m Model) helpLine() string {
	return fmt.Sprintf(
		"%s:new  %s:edit  %s:log  %s/%s/%s:start/stop/restart  %s:toggle  %s:delete  /:filter  %s:refresh  %s:quit",
		m.keys.New, m.keys.Edit, m.keys.Log,
		m.keys.Start, m.keys.Stop, m.keys.Restart,
		m.keys.Toggle, m.keys.Delete,
		m.keys.Refresh, m.keys.Quit,
	)
}

func (m Model) loadCmd() tea.Cmd {
	loader := m.loader
	return func() tea.Msg {
		if loader == nil {
			return JobsLoadedMsg{Err: fmt.Errorf("loader not configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
		defer cancel()
		res, err := loader.List(ctx)
		return JobsLoadedMsg{Result: res, Err: err}
	}
}

// tickCmd schedules one auto-refresh tick after m.interval. The tick
// triggers a reload + re-schedules itself from the Update handler.
func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(m.interval, func(time.Time) tea.Msg { return autoRefreshMsg{} })
}
