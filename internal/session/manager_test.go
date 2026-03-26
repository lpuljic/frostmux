package session

import (
	"testing"
)

func TestExpandEditor(t *testing.T) {
	tests := []struct {
		name   string
		cmd    string
		editor string
		visual string
		want   string
	}{
		{
			name:   "replaces $EDITOR with env value",
			cmd:    "$EDITOR .",
			editor: "nvim",
			want:   "nvim .",
		},
		{
			name:   "$VISUAL takes priority over $EDITOR",
			cmd:    "$EDITOR .",
			editor: "vim",
			visual: "code",
			want:   "code .",
		},
		{
			name:   "replaces $VISUAL directly",
			cmd:    "$VISUAL .",
			visual: "code",
			want:   "code .",
		},
		{
			name: "falls back to vi when both are empty",
			cmd:  "$EDITOR .",
			want: "vi .",
		},
		{
			name: "leaves commands without $EDITOR alone",
			cmd:  "go test ./...",
			want: "go test ./...",
		},
		{
			name:   "handles $EDITOR mid-string",
			cmd:    "exec $EDITOR --wait .",
			editor: "nvim",
			want:   "exec nvim --wait .",
		},
		{
			name:   "replaces both $EDITOR and $VISUAL in same command",
			cmd:    "$VISUAL or $EDITOR",
			editor: "vim",
			visual: "code",
			want:   "code or code",
		},
		{
			name: "empty command passthrough",
			cmd:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("EDITOR", tt.editor)
			t.Setenv("VISUAL", tt.visual)

			got := expandEditor(tt.cmd)
			if got != tt.want {
				t.Errorf("expandEditor(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestParseFocus(t *testing.T) {
	tests := []struct {
		name       string
		focus      string
		defaultWin string
		wantWin    string
		wantPane   int
	}{
		{
			name:       "empty focus returns default",
			focus:      "",
			defaultWin: "editor",
			wantWin:    "editor",
			wantPane:   0,
		},
		{
			name:       "window only",
			focus:      "code",
			defaultWin: "editor",
			wantWin:    "code",
			wantPane:   0,
		},
		{
			name:       "window and pane",
			focus:      "code.2",
			defaultWin: "editor",
			wantWin:    "code",
			wantPane:   2,
		},
		{
			name:       "invalid pane index falls back to 0",
			focus:      "code.abc",
			defaultWin: "editor",
			wantWin:    "code",
			wantPane:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWin, gotPane := parseFocus(tt.focus, tt.defaultWin)
			if gotWin != tt.wantWin {
				t.Errorf("parseFocus(%q, %q) win = %q, want %q", tt.focus, tt.defaultWin, gotWin, tt.wantWin)
			}
			if gotPane != tt.wantPane {
				t.Errorf("parseFocus(%q, %q) pane = %d, want %d", tt.focus, tt.defaultWin, gotPane, tt.wantPane)
			}
		})
	}
}
