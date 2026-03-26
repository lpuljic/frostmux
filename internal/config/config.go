package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Session string   `yaml:"session"`
	Focus   string   `yaml:"focus,omitempty"`
	Windows []Window `yaml:"windows"`
}

type Window struct {
	Name   string
	Root   string
	Layout string
	Panes  []Pane
}

type Pane struct {
	Command string `yaml:"command"`
	Root    string `yaml:"root,omitempty"`
}

// windowFull is the verbose YAML form with root/layout/panes.
// Only used for (un)marshaling, not passed around elsewhere.
type windowFull struct {
	Root   string `yaml:"root,omitempty"`
	Layout string `yaml:"layout,omitempty"`
	Panes  []Pane `yaml:"panes"`
}

func Parse(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Session == "" {
		base := filepath.Base(path)
		cfg.Session = strings.TrimSuffix(base, filepath.Ext(base))
	}

	home, _ := os.UserHomeDir()
	for i := range cfg.Windows {
		if cfg.Windows[i].Root == "" {
			cfg.Windows[i].Root = home
		} else {
			cfg.Windows[i].Root = ExpandPath(cfg.Windows[i].Root)
		}
	}

	return &cfg, nil
}

// UnmarshalYAML handles the three config flavors:
//
//   - editor: nvim           # scalar (single pane)
//   - logs:                   # sequence (multiple panes)
//   - tail -f app.log
//   - tail -f error.log
//   - code:                   # mapping (full control)
//     root: ~/src
//     layout: main-vertical
//     panes: [...]
func (w *Window) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode || len(node.Content) < 2 {
		return fmt.Errorf("window must be a mapping with a name key")
	}

	// the window name is always the first key in the mapping
	w.Name = node.Content[0].Value
	val := node.Content[1]

	switch val.Kind {
	case yaml.ScalarNode:
		w.Panes = []Pane{{Command: val.Value}}

	case yaml.SequenceNode:
		for _, item := range val.Content {
			switch item.Kind {
			case yaml.ScalarNode:
				w.Panes = append(w.Panes, Pane{Command: item.Value})
			case yaml.MappingNode:
				var p Pane
				if err := item.Decode(&p); err != nil {
					return fmt.Errorf("parsing pane in window %q: %w", w.Name, err)
				}
				w.Panes = append(w.Panes, p)
			}
		}

	case yaml.MappingNode:
		var full windowFull
		if err := val.Decode(&full); err != nil {
			return fmt.Errorf("parsing window %q: %w", w.Name, err)
		}
		w.Root = full.Root
		w.Layout = full.Layout
		w.Panes = full.Panes
	}

	if len(w.Panes) == 0 {
		w.Panes = []Pane{{}}
	}

	return nil
}

// MarshalYAML picks the most compact YAML representation that still
// roundtrips correctly. Single pane with no root? Scalar. Multiple simple
// panes? Sequence. Anything else gets the full mapping form.
func (w Window) MarshalYAML() (any, error) {
	home, _ := os.UserHomeDir()
	isHome := w.Root == "" || w.Root == home || w.Root == "~"

	if isHome && len(w.Panes) == 1 && w.Layout == "" && w.Panes[0].Root == "" {
		return map[string]string{w.Name: w.Panes[0].Command}, nil
	}

	if isHome && w.Layout == "" && allPanesSimple(w.Panes) {
		cmds := make([]string, len(w.Panes))
		for i, p := range w.Panes {
			cmds[i] = p.Command
		}
		return map[string][]string{w.Name: cmds}, nil
	}

	root := w.Root
	if isHome {
		root = ""
	}

	return map[string]windowFull{
		w.Name: {
			Root:   root,
			Layout: w.Layout,
			Panes:  w.Panes,
		},
	}, nil
}

func allPanesSimple(panes []Pane) bool {
	for _, p := range panes {
		if p.Root != "" {
			return false
		}
	}
	return true
}

// Dir returns the config directory, respecting FROSTMUX_CONFIG and
// XDG_CONFIG_HOME before falling back to ~/.config/frostmux.
func Dir() string {
	if dir := os.Getenv("FROSTMUX_CONFIG"); dir != "" {
		return dir
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "frostmux")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "frostmux")
}

// FindConfig tries .yml first, then .yaml, because life's too short
// to argue about file extensions.
func FindConfig(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("no project specified")
	}

	path := filepath.Join(Dir(), name+".yml")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	path = filepath.Join(Dir(), name+".yaml")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("config %q not found in %s", name, Dir())
}

// ListConfigs returns the names of all config files in the config directory.
func ListConfigs() ([]string, error) {
	dir := Dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if before, ok := strings.CutSuffix(name, ".yml"); ok {
			names = append(names, before)
		} else if before, ok := strings.CutSuffix(name, ".yaml"); ok {
			names = append(names, before)
		}
	}
	return names, nil
}

func ExpandPath(path string) string {
	if path == "" {
		return path
	}
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func CompactPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == home {
		return "~"
	}
	if after, ok := strings.CutPrefix(path, home+"/"); ok {
		return "~/" + after
	}
	return path
}

func ResolveRoot(base, relative string) string {
	if relative == "" {
		return base
	}
	relative = ExpandPath(relative)
	if filepath.IsAbs(relative) {
		return relative
	}
	return filepath.Join(base, relative)
}
