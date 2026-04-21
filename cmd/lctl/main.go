package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/lctl/internal/config"
	"github.com/flexphere/lctl/internal/launchd"
	"github.com/flexphere/lctl/internal/plist"
	"github.com/flexphere/lctl/internal/service"
	"github.com/flexphere/lctl/internal/tui"
	"github.com/flexphere/lctl/internal/tui/common"
	"github.com/flexphere/lctl/internal/tui/dashboard"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "cron":
			os.Exit(cronSubcommand(args[1:], os.Stdout, os.Stderr))
		case "--help", "-h", "help":
			printUsage(os.Stdout)
			return
		}
	}
	if err := runTUI(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printUsage(w *os.File) {
	_, _ = fmt.Fprintln(w, `lctl — launchd agents TUI

Usage:
  lctl                      Launch the TUI
  lctl cron [--inline] EXPR Print a schedule: YAML block from a cron expression
  lctl help                 Show this message

Configuration:
  ~/.config/lctl/config.toml    runtime settings (auto-refresh, keymap, colors)
  ~/.config/lctl/templates/     YAML templates for new agents
  ~/.config/lctl/scripts/       location for slash-less 'program:' values`)
}

func runTUI() error {
	paths, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("resolve config paths: %w", err)
	}
	if err := config.Ensure(paths); err != nil {
		return fmt.Errorf("prepare config dirs: %w", err)
	}
	settings, err := config.LoadSettings(paths)
	if err != nil {
		// Surface the issue but continue with defaults so a broken
		// config.toml never locks the user out of their agents.
		fmt.Fprintf(os.Stderr, "lctl: config: %v (using defaults)\n", err)
	}
	common.Apply(common.NewTheme(settings.List))

	store, err := plist.NewUserStore()
	if err != nil {
		return fmt.Errorf("init plist store: %w", err)
	}
	client, err := launchd.New()
	if err != nil {
		return fmt.Errorf("init launchctl: %w", err)
	}
	svc := service.New(store, client)
	ops := service.NewOps(store, client)
	flow := &dashboard.EditFlow{
		Store:      store,
		Client:     client,
		ScriptsDir: paths.Scripts,
	}
	dash := dashboard.Settings{
		Keymap:          settings.Keymap,
		AutoRefresh:     settings.List.AutoRefresh,
		RefreshInterval: time.Duration(settings.List.RefreshInterval) * time.Second,
		Layout:          settings.List.Layout,
		Variables:       settings.List.Variables,
		Filter:          settings.Filter,
	}

	p := tea.NewProgram(tui.New(svc, ops, flow, paths.Templates, dash, settings.Templates), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
