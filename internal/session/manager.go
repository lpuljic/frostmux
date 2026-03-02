package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lpuljic/muxify/internal/config"
	"github.com/lpuljic/muxify/internal/tmux"
)

type Manager struct {
	tmux *tmux.Client
}

func NewManager(t *tmux.Client) *Manager {
	return &Manager{tmux: t}
}

func (m *Manager) Start(cfg *config.Config) error {
	if len(cfg.Windows) == 0 {
		return fmt.Errorf("config has no windows defined")
	}

	if m.tmux.HasSession(cfg.Session) {
		return m.attach(cfg.Session)
	}

	for i, win := range cfg.Windows {
		winRoot := win.Root

		firstPaneRoot := winRoot
		if len(win.Panes) > 0 && win.Panes[0].Root != "" {
			firstPaneRoot = config.ResolveRoot(winRoot, win.Panes[0].Root)
		}

		if i == 0 {
			if err := m.tmux.NewSession(cfg.Session, firstPaneRoot); err != nil {
				return fmt.Errorf("creating session %q: %w", cfg.Session, err)
			}
			if win.Name != "" {
				m.tmux.RenameWindow(cfg.Session, win.Name)
			}
		} else {
			if err := m.tmux.NewWindow(cfg.Session, win.Name, firstPaneRoot); err != nil {
				return fmt.Errorf("creating window %q: %w", win.Name, err)
			}
		}

		winTarget := cfg.Session + ":" + win.Name

		for j := 1; j < len(win.Panes); j++ {
			paneRoot := config.ResolveRoot(winRoot, win.Panes[j].Root)
			if err := m.tmux.SplitWindow(winTarget, paneRoot); err != nil {
				return fmt.Errorf("splitting pane in window %q: %w", win.Name, err)
			}
		}

		if win.Layout != "" {
			m.tmux.SelectLayout(winTarget, win.Layout)
		}

		for j, pane := range win.Panes {
			if pane.Command != "" {
				paneTarget := fmt.Sprintf("%s.%d", winTarget, j)
				m.tmux.SendKeys(paneTarget, pane.Command)
			}
		}
	}

	firstWin := cfg.Session + ":" + cfg.Windows[0].Name
	m.tmux.SelectWindow(firstWin)
	m.tmux.SelectPane(firstWin + ".0")

	return m.attach(cfg.Session)
}

func (m *Manager) Stop(name string) error {
	if !m.tmux.HasSession(name) {
		return fmt.Errorf("session %q not found", name)
	}
	return m.tmux.KillSession(name)
}

func (m *Manager) Freeze(saveName string) (*config.Config, error) {
	session, err := m.tmux.CurrentSession()
	if err != nil {
		return nil, err
	}

	windows, err := m.tmux.ListWindows(session)
	if err != nil {
		return nil, fmt.Errorf("listing windows: %w", err)
	}

	name := saveName
	if name == "" {
		name = session
	}

	cfg := &config.Config{
		Session: name,
	}

	for _, win := range windows {
		target := fmt.Sprintf("%s:%s", session, win.Index)
		panes, err := m.tmux.ListPanes(target)
		if err != nil {
			continue
		}

		w := config.Window{
			Name: win.Name,
		}

		if len(panes) > 1 {
			w.Layout = win.Layout
		}

		if len(panes) > 0 {
			w.Root = config.CompactPath(panes[0].CurrentPath)
		}

		for _, p := range panes {
			pane := config.Pane{}

			expandedWinRoot := config.ExpandPath(w.Root)
			if p.CurrentPath != expandedWinRoot {
				pane.Root = config.CompactPath(p.CurrentPath)
			}

			w.Panes = append(w.Panes, pane)
		}

		cfg.Windows = append(cfg.Windows, w)
	}

	return cfg, nil
}

func (m *Manager) attach(name string) error {
	if tmux.InsideTmux() {
		return m.tmux.SwitchClient(name)
	}
	return m.tmux.Attach(name)
}

func isShell(cmd string) bool {
	shells := map[string]bool{
		"bash": true, "zsh": true, "fish": true,
		"sh": true, "dash": true, "ksh": true,
		"tcsh": true, "csh": true,
	}
	return shells[filepath.Base(cmd)]
}

func ListConfigs() ([]string, error) {
	dir := config.Dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var configs []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".yml") {
			configs = append(configs, strings.TrimSuffix(name, ".yml"))
		} else if strings.HasSuffix(name, ".yaml") {
			configs = append(configs, strings.TrimSuffix(name, ".yaml"))
		}
	}
	return configs, nil
}
