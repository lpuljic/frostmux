package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseShorthandString(t *testing.T) {
	cfg := parseFromYAML(t, `
session: test
windows:
  - editor: nvim
`)

	if len(cfg.Windows) != 1 {
		t.Fatalf("expected 1 window, got %d", len(cfg.Windows))
	}

	w := cfg.Windows[0]
	if w.Name != "editor" {
		t.Errorf("name = %q, want %q", w.Name, "editor")
	}
	if len(w.Panes) != 1 {
		t.Fatalf("expected 1 pane, got %d", len(w.Panes))
	}
	if w.Panes[0].Command != "nvim" {
		t.Errorf("command = %q, want %q", w.Panes[0].Command, "nvim")
	}
}

func TestParseShorthandList(t *testing.T) {
	cfg := parseFromYAML(t, `
session: test
windows:
  - logs:
      - tail -f app.log
      - tail -f error.log
`)

	if len(cfg.Windows) != 1 {
		t.Fatalf("expected 1 window, got %d", len(cfg.Windows))
	}

	w := cfg.Windows[0]
	if w.Name != "logs" {
		t.Errorf("name = %q, want %q", w.Name, "logs")
	}
	if len(w.Panes) != 2 {
		t.Fatalf("expected 2 panes, got %d", len(w.Panes))
	}
	if w.Panes[0].Command != "tail -f app.log" {
		t.Errorf("pane[0] command = %q, want %q", w.Panes[0].Command, "tail -f app.log")
	}
	if w.Panes[1].Command != "tail -f error.log" {
		t.Errorf("pane[1] command = %q, want %q", w.Panes[1].Command, "tail -f error.log")
	}
}

func TestParseFullForm(t *testing.T) {
	cfg := parseFromYAML(t, `
session: test
windows:
  - code:
      root: /tmp/src
      layout: main-vertical
      panes:
        - command: nvim
        - command: go test
          root: /tmp/tests
`)

	if len(cfg.Windows) != 1 {
		t.Fatalf("expected 1 window, got %d", len(cfg.Windows))
	}

	w := cfg.Windows[0]
	if w.Name != "code" {
		t.Errorf("name = %q, want %q", w.Name, "code")
	}
	if w.Root != "/tmp/src" {
		t.Errorf("root = %q, want %q", w.Root, "/tmp/src")
	}
	if w.Layout != "main-vertical" {
		t.Errorf("layout = %q, want %q", w.Layout, "main-vertical")
	}
	if len(w.Panes) != 2 {
		t.Fatalf("expected 2 panes, got %d", len(w.Panes))
	}
	if w.Panes[0].Command != "nvim" {
		t.Errorf("pane[0] command = %q, want %q", w.Panes[0].Command, "nvim")
	}
	if w.Panes[1].Command != "go test" {
		t.Errorf("pane[1] command = %q, want %q", w.Panes[1].Command, "go test")
	}
	if w.Panes[1].Root != "/tmp/tests" {
		t.Errorf("pane[1] root = %q, want %q", w.Panes[1].Root, "/tmp/tests")
	}
}

func TestParseMixedWindows(t *testing.T) {
	cfg := parseFromYAML(t, `
session: mixed
windows:
  - editor: nvim
  - logs:
      - tail -f app.log
      - tail -f error.log
  - dev:
      root: ~/backend
      layout: tiled
      panes:
        - command: go run .
        - command: go test ./...
          root: ~/backend/tests
`)

	if len(cfg.Windows) != 3 {
		t.Fatalf("expected 3 windows, got %d", len(cfg.Windows))
	}

	if cfg.Windows[0].Name != "editor" || cfg.Windows[0].Panes[0].Command != "nvim" {
		t.Error("string shorthand parsed incorrectly")
	}

	if cfg.Windows[1].Name != "logs" || len(cfg.Windows[1].Panes) != 2 {
		t.Error("list shorthand parsed incorrectly")
	}

	home, _ := os.UserHomeDir()
	wantRoot := filepath.Join(home, "backend")
	if cfg.Windows[2].Name != "dev" || cfg.Windows[2].Layout != "tiled" || cfg.Windows[2].Root != wantRoot {
		t.Errorf("full form parsed incorrectly, root = %q, want %q", cfg.Windows[2].Root, wantRoot)
	}
}

func TestWindowRootDefaultsToHome(t *testing.T) {
	cfg := parseFromYAML(t, `
session: test
windows:
  - shell: ""
`)

	home, _ := os.UserHomeDir()
	if cfg.Windows[0].Root != home {
		t.Errorf("root = %q, want %q", cfg.Windows[0].Root, home)
	}
}

func TestWindowRootExpands(t *testing.T) {
	cfg := parseFromYAML(t, `
session: test
windows:
  - code:
      root: ~/projects
      panes:
        - command: nvim
`)

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, "projects")
	if cfg.Windows[0].Root != want {
		t.Errorf("root = %q, want %q", cfg.Windows[0].Root, want)
	}
}

func TestSessionFallbackToFilename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my-project.yml")

	if err := os.WriteFile(path, []byte(`
windows:
  - shell: ""
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Session != "my-project" {
		t.Errorf("session = %q, want %q", cfg.Session, "my-project")
	}
}

func TestEmptyPaneFallback(t *testing.T) {
	cfg := parseFromYAML(t, `
session: test
windows:
  - shell: ""
`)

	if len(cfg.Windows[0].Panes) != 1 {
		t.Fatalf("expected 1 pane, got %d", len(cfg.Windows[0].Panes))
	}
	if cfg.Windows[0].Panes[0].Command != "" {
		t.Errorf("command = %q, want empty", cfg.Windows[0].Panes[0].Command)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"tilde prefix", "~/code", filepath.Join(home, "code")},
		{"tilde only", "~", home},
		{"absolute", "/tmp/foo", "/tmp/foo"},
		{"relative", "foo/bar", "foo/bar"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandPath(tt.input)
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveRoot(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		relative string
		want     string
	}{
		{"empty relative", "/home/user/project", "", "/home/user/project"},
		{"relative path", "/home/user/project", "src", "/home/user/project/src"},
		{"dot relative", "/home/user/project", "./src", "/home/user/project/src"},
		{"absolute override", "/home/user/project", "/tmp/other", "/tmp/other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveRoot(tt.base, tt.relative)
			if got != tt.want {
				t.Errorf("ResolveRoot(%q, %q) = %q, want %q", tt.base, tt.relative, got, tt.want)
			}
		})
	}
}

func TestCompactPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"home dir", home, "~"},
		{"subdir", home + "/code/project", "~/code/project"},
		{"absolute non-home", "/tmp/foo", "/tmp/foo"},
		{"partial match", home + "extra/path", home + "extra/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompactPath(tt.input)
			if got != tt.want {
				t.Errorf("CompactPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSequenceWithPaneObjects(t *testing.T) {
	cfg := parseFromYAML(t, `
session: test
windows:
  - code:
      - command: nvim
        root: /tmp/src
      - command: go test
        root: /tmp/tests
`)

	w := cfg.Windows[0]
	if len(w.Panes) != 2 {
		t.Fatalf("expected 2 panes, got %d", len(w.Panes))
	}
	if w.Panes[0].Root != "/tmp/src" {
		t.Errorf("pane[0] root = %q, want %q", w.Panes[0].Root, "/tmp/src")
	}
	if w.Panes[1].Root != "/tmp/tests" {
		t.Errorf("pane[1] root = %q, want %q", w.Panes[1].Root, "/tmp/tests")
	}
}

func parseFromYAML(t *testing.T, data string) *Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yml")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}
