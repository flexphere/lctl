package common

import (
	"testing"

	"github.com/flexphere/lctl/internal/config"
)

func TestDefaultThemePopulatesAllStyles(t *testing.T) {
	th := DefaultTheme()
	if th.Title.GetBold() == false {
		t.Error("title should be bold by default")
	}
	if th.HeaderText.GetForeground() == nil {
		t.Error("header text should have a default foreground")
	}
}

func TestApplyRebindsGlobals(t *testing.T) {
	custom := NewTheme(config.ListSettings{
		Brand:  config.BrandSettings{Color: "#abcdef"},
		Help:   config.HelpSettings{Color: "200"},
		Header: config.HeaderSettings{Color: "#ffffff"},
		Status: config.StatusColors{Running: "201", Idle: "202", Errored: "203"},
	})
	Apply(custom)
	defer Apply(DefaultTheme())
	if TitleStyle.GetForeground() != custom.Title.GetForeground() {
		t.Errorf("Apply did not update TitleStyle")
	}
}
