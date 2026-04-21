package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/lctl/internal/config"
	"github.com/flexphere/lctl/internal/service"
)

func TestFilterPromptConfigurable(t *testing.T) {
	settings := Settings{
		Filter: config.FilterSettings{
			Prompt:      "❯ ",
			PromptColor: "#bb9af7",
			TextColor:   "#c0caf5",
		},
	}
	jobs := []service.Job{
		{Label: "lctl.com.flexphere.alpha"},
	}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, nil, nil, settings)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})

	// Typing state: list.FilterInput should receive the configured prompt.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if m.list.FilterInput.Prompt != "❯ " {
		t.Errorf("FilterInput.Prompt override not applied: %q", m.list.FilterInput.Prompt)
	}
	out := m.View()
	if !strings.Contains(out, "❯") {
		t.Errorf("configured prompt char missing from view:\n%s", out)
	}
}

func TestFilterAppliedUsesPromptAndText(t *testing.T) {
	settings := Settings{
		Filter: config.FilterSettings{
			Prompt:      "? ",
			PromptColor: "#bb9af7",
			TextColor:   "#c0caf5",
		},
	}
	jobs := []service.Job{
		{Label: "lctl.com.flexphere.alpha"},
		{Label: "lctl.com.flexphere.beta"},
	}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, nil, nil, settings)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m, _ = m.Update(JobsLoadedMsg{Result: service.ListResult{Jobs: jobs}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "beta" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	line := m.itemStatusLine()
	if !strings.Contains(line, "? ") {
		t.Errorf("applied prompt prefix missing: %q", line)
	}
	if !strings.Contains(line, "beta") {
		t.Errorf("applied filter value missing: %q", line)
	}
}

func TestFilterDefaultPromptIsSlashSpace(t *testing.T) {
	jobs := []service.Job{{Label: "lctl.com.x"}}
	m := New(stubLoader{res: service.ListResult{Jobs: jobs}}, nil, nil, Settings{})
	if got := m.list.FilterInput.Prompt; got != "/ " {
		t.Errorf("default FilterInput.Prompt should be \"/ \", got %q", got)
	}
}
