package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Client struct {
	bin string
}

func NewClient() *Client {
	bin := "tmux"
	if path, err := exec.LookPath("tmux"); err == nil {
		bin = path
	}
	return &Client{bin: bin}
}

func (c *Client) exec(args ...string) (string, error) {
	cmd := exec.Command(c.bin, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (c *Client) NewSession(name, root string) error {
	_, err := c.exec("new-session", "-d", "-s", name, "-c", root)
	return err
}

func (c *Client) RenameWindow(sessionTarget, name string) error {
	_, err := c.exec("rename-window", "-t", sessionTarget, name)
	return err
}

func (c *Client) NewWindow(session, name, root string) error {
	_, err := c.exec("new-window", "-t", session, "-n", name, "-c", root)
	return err
}

func (c *Client) SplitWindow(target, root string) error {
	_, err := c.exec("split-window", "-t", target, "-c", root)
	return err
}

func (c *Client) SendKeys(target, keys string) error {
	_, err := c.exec("send-keys", "-t", target, keys, "Enter")
	return err
}

func (c *Client) SelectLayout(target, layout string) error {
	_, err := c.exec("select-layout", "-t", target, layout)
	return err
}

func (c *Client) SelectWindow(target string) error {
	_, err := c.exec("select-window", "-t", target)
	return err
}

func (c *Client) SelectPane(target string) error {
	_, err := c.exec("select-pane", "-t", target)
	return err
}

func (c *Client) Attach(name string) error {
	cmd := exec.Command(c.bin, "attach-session", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *Client) SwitchClient(name string) error {
	_, err := c.exec("switch-client", "-t", name)
	return err
}

func (c *Client) HasSession(name string) bool {
	_, err := c.exec("has-session", "-t", name)
	return err == nil
}

func (c *Client) KillSession(name string) error {
	_, err := c.exec("kill-session", "-t", name)
	return err
}

func (c *Client) ListSessions() ([]string, error) {
	out, err := c.exec("list-sessions", "-F", "#{session_name}")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

func (c *Client) CurrentSession() (string, error) {
	out, err := c.exec("display-message", "-p", "#{session_name}")
	if err != nil {
		return "", fmt.Errorf("not inside a tmux session")
	}
	return out, nil
}

type WindowInfo struct {
	Index  string
	Name   string
	Layout string
}

type PaneInfo struct {
	Index          string
	CurrentPath    string
	CurrentCommand string
}

func (c *Client) ListWindows(session string) ([]WindowInfo, error) {
	out, err := c.exec("list-windows", "-t", session, "-F", "#{window_index}\t#{window_name}\t#{window_layout}")
	if err != nil {
		return nil, err
	}

	var windows []WindowInfo
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		windows = append(windows, WindowInfo{
			Index:  parts[0],
			Name:   parts[1],
			Layout: parts[2],
		})
	}
	return windows, nil
}

func (c *Client) ListPanes(target string) ([]PaneInfo, error) {
	out, err := c.exec("list-panes", "-t", target, "-F", "#{pane_index}\t#{pane_current_path}\t#{pane_current_command}")
	if err != nil {
		return nil, err
	}

	var panes []PaneInfo
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		panes = append(panes, PaneInfo{
			Index:          parts[0],
			CurrentPath:    parts[1],
			CurrentCommand: parts[2],
		})
	}
	return panes, nil
}

func InsideTmux() bool {
	return os.Getenv("TMUX") != ""
}
