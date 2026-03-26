package session

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lpuljic/frostmux/internal/config"
	"github.com/lpuljic/frostmux/internal/tmux"
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

	// already running? just jump to it
	if m.tmux.HasSession(cfg.Session) {
		return m.Attach(cfg.Session)
	}

	for i, win := range cfg.Windows {
		winRoot := win.Root

		// tmux new-session / new-window needs a starting dir for the first pane,
		// the rest get their dirs when we split
		firstPaneRoot := winRoot
		if len(win.Panes) > 0 && win.Panes[0].Root != "" {
			firstPaneRoot = config.ResolveRoot(winRoot, win.Panes[0].Root)
		}

		if i == 0 {
			// first window is created implicitly with the session
			if err := m.tmux.NewSession(cfg.Session, firstPaneRoot); err != nil {
				return fmt.Errorf("creating session %q: %w", cfg.Session, err)
			}
			if win.Name != "" {
				if err := m.tmux.RenameWindow(cfg.Session, win.Name); err != nil {
					return fmt.Errorf("renaming window in %q: %w", cfg.Session, err)
				}
			}
		} else {
			if err := m.tmux.NewWindow(cfg.Session, win.Name, firstPaneRoot); err != nil {
				return fmt.Errorf("creating window %q: %w", win.Name, err)
			}
		}

		winTarget := cfg.Session + ":" + win.Name

		// pane 0 already exists from the window creation, start splitting from 1
		for j := 1; j < len(win.Panes); j++ {
			paneRoot := config.ResolveRoot(winRoot, win.Panes[j].Root)
			if err := m.tmux.SplitWindow(winTarget, paneRoot); err != nil {
				return fmt.Errorf("splitting pane in window %q: %w", win.Name, err)
			}
		}

		if win.Layout != "" {
			if err := m.tmux.SelectLayout(winTarget, win.Layout); err != nil {
				return fmt.Errorf("setting layout for window %q: %w", win.Name, err)
			}
		}

		// fire off startup commands (send-keys, not shell -c, so the
		// command shows up in the pane's scrollback naturally)
		for j, pane := range win.Panes {
			if pane.Command != "" {
				paneTarget := fmt.Sprintf("%s.%d", winTarget, j)
				if err := m.tmux.SendKeys(paneTarget, expandEditor(pane.Command)); err != nil {
					return fmt.Errorf("sending command to %s: %w", paneTarget, err)
				}
			}
		}
	}

	focusWin, focusPane := parseFocus(cfg.Focus, cfg.Windows[0].Name)
	target := cfg.Session + ":" + focusWin
	if err := m.tmux.SelectWindow(target); err != nil {
		return fmt.Errorf("selecting window %q: %w", focusWin, err)
	}
	if err := m.tmux.SelectPane(fmt.Sprintf("%s.%d", target, focusPane)); err != nil {
		return fmt.Errorf("selecting pane %d in %q: %w", focusPane, focusWin, err)
	}

	return m.Attach(cfg.Session)
}

func (m *Manager) Stop(name string) error {
	if !m.tmux.HasSession(name) {
		return fmt.Errorf("session %q not found", name)
	}
	return m.tmux.KillSession(name)
}

func (m *Manager) Freeze() (*config.Config, error) {
	session, err := m.tmux.CurrentSession()
	if err != nil {
		return nil, fmt.Errorf("not inside a tmux session, nothing to freeze")
	}

	windows, err := m.tmux.ListWindows(session)
	if err != nil {
		return nil, fmt.Errorf("listing windows: %w", err)
	}

	cfg := &config.Config{
		Session: session,
	}

	// capture which window/pane the user is currently looking at
	if activeWin, err := m.tmux.ActiveWindow(session); err == nil {
		activePaneIdx, _ := m.tmux.ActivePane(session)
		if activePaneIdx != "" && activePaneIdx != "0" {
			cfg.Focus = activeWin + "." + activePaneIdx
		} else if len(windows) > 0 && activeWin != windows[0].Name {
			cfg.Focus = activeWin
		}
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

// Attach or switch depending on whether we're already inside tmux.
// switch-client works from within tmux, attach-session from outside.
func (m *Manager) Attach(name string) error {
	if tmux.InsideTmux() {
		return m.tmux.SwitchClient(name)
	}
	return m.tmux.Attach(name)
}

// parseFocus splits a focus string like "code" or "code.1" into a
// window name and pane index. Falls back to defaultWin and pane 0.
func parseFocus(focus, defaultWin string) (string, int) {
	if focus == "" {
		return defaultWin, 0
	}

	win, paneStr, ok := strings.Cut(focus, ".")
	if !ok {
		return win, 0
	}

	pane, err := strconv.Atoi(paneStr)
	if err != nil {
		return win, 0
	}
	return win, pane
}

// expandEditor resolves $EDITOR/$VISUAL before sending commands to tmux
// panes, since the shell inside tmux might not have them set. Falls back to vi.
func expandEditor(cmd string) string {
	if !strings.Contains(cmd, "$EDITOR") && !strings.Contains(cmd, "$VISUAL") {
		return cmd
	}

	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}

	cmd = strings.ReplaceAll(cmd, "$VISUAL", editor)
	cmd = strings.ReplaceAll(cmd, "$EDITOR", editor)
	return cmd
}
