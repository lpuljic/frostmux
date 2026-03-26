<p align="center">
  <img src="logo.png" alt="frostmux" width="200">
</p>

<p align="center">A tmux session manager that writes your config for you.</p>

**Capture first, config second.**

Arrange your tmux windows and panes the way you like them, freeze the layout to YAML, and replay it whenever you need it. No hand-written configs required.

## Requirements

- [tmux](https://github.com/tmux/tmux)
- [Go 1.25+](https://go.dev/dl/) (for installation)

## Install

```bash
go install github.com/lpuljic/frostmux/cmd/frostmux@latest
```

## Quick Start

```bash
# 1. Create a new tmux session
frostmux new my-project

# 2. Set up your windows, panes, working directories however you want
#    ... do your thing ...

# 3. Freeze it
frostmux freeze

# 4. Next time, just start it
frostmux my-project
```

That's the whole workflow. Set up once, replay forever.

## Commands

```
frostmux <project>              Start or attach to a session
frostmux new <project>          Create a new tmux session
frostmux freeze                 Capture current session to YAML
frostmux stop <project>         Kill a tmux session
frostmux list                   List available configs
frostmux edit <project>         Edit an existing config
frostmux delete <project>       Delete a config
```

### `frostmux new <project>`

Creates a fresh tmux session and attaches to it. If a session with that name already exists, it just attaches.

```bash
frostmux new api
```

### `frostmux freeze`

Captures your current tmux session (windows, panes, working directories, layout) and saves it as YAML. Must be run from inside tmux.

```bash
frostmux freeze    # saves to ~/.config/frostmux/<session-name>.yml
```

The config file name comes from your tmux session name. Running freeze again overwrites the previous config with the current state.

### `frostmux <project>`

Starts a session from a saved config. If the session is already running, it attaches instead of creating a duplicate.

```bash
frostmux my-project
```

## Config Format

Configs live in `~/.config/frostmux/` (override with `$FROSTMUX_CONFIG` or `$XDG_CONFIG_HOME`).

You'll rarely write these by hand since `freeze` generates them, but here's what they look like:

### Shorthand: single command per window

```yaml
session: api
windows:
  - editor: nvim
  - server: go run ./cmd/server
```

Windows without a `root` default to `~`.

### Multi-pane shorthand

```yaml
windows:
  - logs:
      - tail -f app.log
      - tail -f error.log
```

### Full form: per-pane control

```yaml
windows:
  - code:
      root: ~/code/api
      layout: main-vertical
      panes:
        - command: nvim
        - command: go test ./...
          root: ~/code/api/tests
```

### Focus

By default frostmux selects the first window, first pane. Use `focus` to land somewhere else:

```yaml
session: my-project
focus: dev       # land on the "dev" window, pane 0
windows:
  - editor: nvim
  - dev:
      root: ~/code/my-project
      panes:
        - command: go run .
        - command: go test ./...
```

Target a specific pane with `window.pane`:

```yaml
focus: dev.1     # land on the "dev" window, second pane
```

When you `freeze` a session, frostmux captures whichever window and pane you're currently looking at.

## Shell Completion

```bash
# zsh (~/.zshrc)
eval "$(frostmux completion zsh)"

# bash (~/.bashrc)
eval "$(frostmux completion bash)"

# fish (~/.config/fish/config.fish)
frostmux completion fish | source
```

## Layouts

Standard tmux layouts: `even-horizontal`, `even-vertical`, `main-horizontal`, `main-vertical`, `tiled`.

## Credits

Inspired by [smug](https://github.com/ivaaaan/smug) and [tmuxinator](https://github.com/tmuxinator/tmuxinator).
