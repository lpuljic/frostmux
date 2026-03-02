<p align="center">
  <img src="logo.png" alt="muxify" width="200">
</p>

<h1 align="center">muxify</h1>

<p align="center">A tmux session manager that writes your config for you.</p>

**Capture first, config second.** Arrange your tmux windows and panes the way you like them, freeze the layout to YAML, and replay it whenever you need it. No hand-written configs required.

## Requirements

- [tmux](https://github.com/tmux/tmux)
- [Go 1.21+](https://go.dev/dl/) (for installation)

## Install

```bash
go install github.com/lpuljic/muxify/cmd/muxify@latest
```

## Quick Start

```bash
# 1. Set up tmux however you want, then capture it
muxify freeze my-project

# 2. Next time, replay it
muxify start my-project
# or just
muxify my-project
```

## Usage

### Freeze a running session

Captures your current tmux session (windows, panes, working directories, layout) and saves it as a YAML config.

```bash
muxify freeze my-project    # saves to ~/.config/muxify/my-project.yml
muxify freeze               # prints config to stdout
```

### Start a session

```bash
muxify start my-project
muxify my-project            # shortcut
muxify start -f ./custom.yml # start from a specific file
```

If the session already exists, muxify attaches to it instead of creating a duplicate.

### Scaffold from project detection

```bash
cd ~/code/my-go-project
muxify init
```

Detects your project type and generates a sensible config:

| Detected file    | Windows generated          |
|------------------|----------------------------|
| `go.mod`         | editor, build, test        |
| `package.json`   | editor, dev, test          |
| `Cargo.toml`     | editor, build, test        |
| `Makefile`       | editor, build, shell       |
| (none)           | editor, shell              |

### Other commands

```bash
muxify list              # list saved configs
muxify stop <project>    # kill a session
muxify new <project>     # create a blank config and open in $EDITOR
muxify edit <project>    # edit an existing config
muxify delete <project>  # delete a config
```

## Config Format

Configs live in `~/.config/muxify/` (override with `$MUXIFY_CONFIG` or `$XDG_CONFIG_HOME`).

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

### Mixed: all three in one config

```yaml
session: my-project
windows:
  - editor: nvim
  - notes:
      root: ~/Documents/notes
      panes:
        - command: ""
  - dev:
      root: ~/code/my-project
      layout: tiled
      panes:
        - command: go run .
        - command: go test ./...
```

## Shell Completion

```bash
# zsh (~/.zshrc)
eval "$(muxify completion zsh)"

# bash (~/.bashrc)
eval "$(muxify completion bash)"

# fish (~/.config/fish/config.fish)
muxify completion fish | source
```

## Layouts

Standard tmux layouts: `even-horizontal`, `even-vertical`, `main-horizontal`, `main-vertical`, `tiled`.

## Credits

Inspired by [smug](https://github.com/ivaaaan/smug) and [tmuxinator](https://github.com/tmuxinator/tmuxinator).
