package config

import (
	"os"
	"path/filepath"
)

// Detect sniffs the project directory for known build files and returns
// a config with sensible window presets. Not perfect, but beats starting
// from scratch every time.
func Detect(dir string) *Config {
	dir = ExpandPath(dir)
	if dir == "" || dir == "." {
		dir, _ = os.Getwd()
	}

	root := CompactPath(dir)
	cfg := &Config{
		Session: filepath.Base(dir),
	}

	switch {
	case fileExists(filepath.Join(dir, "go.mod")):
		cfg.Windows = goWindows(root)
	case fileExists(filepath.Join(dir, "package.json")):
		cfg.Windows = nodeWindows(root)
	case fileExists(filepath.Join(dir, "Cargo.toml")):
		cfg.Windows = rustWindows(root)
	case fileExists(filepath.Join(dir, "Makefile")), fileExists(filepath.Join(dir, "makefile")):
		cfg.Windows = makeWindows(root)
	default:
		cfg.Windows = defaultWindows(root)
	}

	return cfg
}

func goWindows(root string) []Window {
	return []Window{
		{Name: "editor", Root: root, Panes: []Pane{{Command: "$EDITOR ."}}},
		{Name: "build", Root: root, Panes: []Pane{{Command: "go build ./..."}}},
		{Name: "test", Root: root, Panes: []Pane{{Command: "go test ./..."}}},
	}
}

func nodeWindows(root string) []Window {
	return []Window{
		{Name: "editor", Root: root, Panes: []Pane{{Command: "$EDITOR ."}}},
		{Name: "dev", Root: root, Panes: []Pane{{Command: "npm run dev"}}},
		{Name: "test", Root: root, Panes: []Pane{{Command: "npm test"}}},
	}
}

func rustWindows(root string) []Window {
	return []Window{
		{Name: "editor", Root: root, Panes: []Pane{{Command: "$EDITOR ."}}},
		{Name: "build", Root: root, Panes: []Pane{{Command: "cargo build"}}},
		{Name: "test", Root: root, Panes: []Pane{{Command: "cargo test"}}},
	}
}

func makeWindows(root string) []Window {
	return []Window{
		{Name: "editor", Root: root, Panes: []Pane{{Command: "$EDITOR ."}}},
		{Name: "build", Root: root, Panes: []Pane{{Command: "make"}}},
		{Name: "shell", Root: root, Panes: []Pane{{}}},
	}
}

func defaultWindows(root string) []Window {
	return []Window{
		{Name: "editor", Root: root, Panes: []Pane{{Command: "$EDITOR ."}}},
		{Name: "shell", Root: root, Panes: []Pane{{}}},
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
