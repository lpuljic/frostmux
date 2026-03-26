package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lpuljic/frostmux/internal/config"
	"github.com/lpuljic/frostmux/internal/session"
	"github.com/lpuljic/frostmux/internal/tmux"
	"gopkg.in/yaml.v3"
)

// set by -ldflags at build time
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	// `frostmux myproject` is the only way to start a session,
	// no redundant "start" subcommand needed
	var err error
	switch cmd {
	case "stop", "st":
		err = cmdStop(args)
	case "list", "ls":
		err = cmdList()
	case "freeze":
		err = cmdFreeze(args)
	case "new", "n":
		err = cmdNew(args)
	case "edit", "e":
		err = cmdEdit(args)
	case "delete", "rm":
		err = cmdDelete(args)
	case "completion":
		err = cmdCompletion(args)
	case "help", "-h", "--help":
		usage()
		return
	case "version", "-v", "--version":
		fmt.Printf("frostmux %s\n", version)
		return
	default:
		err = cmdStart(cmd)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "frostmux: %v\n", err)
		os.Exit(1)
	}
}

func cmdStart(name string) error {
	cfgPath, err := config.FindConfig(name)
	if err != nil {
		return err
	}

	cfg, err := config.Parse(cfgPath)
	if err != nil {
		return err
	}

	mgr := session.NewManager(tmux.NewClient())
	return mgr.Start(cfg)
}

func cmdStop(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: frostmux stop <project>")
	}

	mgr := session.NewManager(tmux.NewClient())
	return mgr.Stop(args[0])
}

func cmdList() error {
	configs, err := config.ListConfigs()
	if err != nil {
		return err
	}

	if len(configs) == 0 {
		fmt.Println("no configs found")
		return nil
	}

	for _, name := range configs {
		fmt.Println(name)
	}
	return nil
}

func cmdFreeze(args []string) error {
	mgr := session.NewManager(tmux.NewClient())
	cfg, err := mgr.Freeze()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	dir := config.Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	path := filepath.Join(dir, cfg.Session+".yml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("frozen %s\n", path)
	return nil
}

func cmdNew(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: frostmux new <project>")
	}

	name := args[0]
	client := tmux.NewClient()
	mgr := session.NewManager(client)

	if client.HasSession(name) {
		return mgr.Attach(name)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	if err := client.NewSession(name, home); err != nil {
		return fmt.Errorf("creating session %q: %w", name, err)
	}

	return mgr.Attach(name)
}

func cmdEdit(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: frostmux edit <project>")
	}

	path, err := config.FindConfig(args[0])
	if err != nil {
		return err
	}

	return openEditor(path)
}

func cmdDelete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: frostmux delete <project>")
	}

	path, err := config.FindConfig(args[0])
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing config: %w", err)
	}

	fmt.Printf("deleted %s\n", path)
	return nil
}

func cmdCompletion(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: frostmux completion <bash|zsh|fish>")
	}

	switch args[0] {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	case "fish":
		fmt.Print(fishCompletion)
	default:
		return fmt.Errorf("unsupported shell %q, use bash, zsh, or fish", args[0])
	}
	return nil
}

const bashCompletion = `_frostmux() {
    local cur prev commands
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    commands="stop list freeze new edit delete completion help version"

    case "$prev" in
        stop|st|edit|e|delete|rm)
            COMPREPLY=($(compgen -W "$(frostmux list 2>/dev/null)" -- "$cur"))
            return
            ;;
        frostmux)
            COMPREPLY=($(compgen -W "$commands $(frostmux list 2>/dev/null)" -- "$cur"))
            return
            ;;
        completion)
            COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur"))
            return
            ;;
    esac
}
complete -F _frostmux frostmux
`

const zshCompletion = `_frostmux() {
    local -a commands projects
    commands=(
        'stop:Kill a tmux session'
        'st:Kill a tmux session'
        'list:List available configs'
        'ls:List available configs'
        'freeze:Capture current session to YAML'
        'new:Create a new tmux session'
        'n:Create a new tmux session'
        'edit:Edit an existing config'
        'e:Edit an existing config'
        'delete:Delete a config'
        'rm:Delete a config'
        'completion:Generate shell completion'
        'help:Show help'
        'version:Print version'
    )

    _arguments '1: :->cmd' '2: :->args'

    case "$state" in
        cmd)
            projects=(${(f)"$(frostmux list 2>/dev/null)"})
            _describe 'command' commands
            [[ ${#projects} -gt 0 ]] && _describe 'project' projects
            ;;
        args)
            case "$words[2]" in
                stop|st|edit|e|delete|rm)
                    projects=(${(f)"$(frostmux list 2>/dev/null)"})
                    [[ ${#projects} -gt 0 ]] && _describe 'project' projects
                    ;;
                completion)
                    _describe 'shell' '(bash zsh fish)'
                    ;;
            esac
            ;;
    esac
}

compdef _frostmux frostmux
`

const fishCompletion = `complete -c frostmux -f
complete -c frostmux -n '__fish_use_subcommand' -a 'stop list freeze new edit delete completion help version'
complete -c frostmux -n '__fish_use_subcommand' -a '(frostmux list 2>/dev/null)'
complete -c frostmux -n '__fish_seen_subcommand_from stop st edit e delete rm' -a '(frostmux list 2>/dev/null)'
complete -c frostmux -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'
`

func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // sorry nano fans
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func usage() {
	fmt.Fprintf(os.Stderr, `frostmux — tmux session manager

Capture first, config second.

Usage:
  frostmux <command> [arguments]
  frostmux <project>              Start or attach to a session

Commands:
  stop, st   <project>            Kill a tmux session
  list, ls                        List available configs
  freeze                          Capture current session to YAML
  new, n     <project>            Create a new tmux session
  edit, e    <project>            Edit an existing config
  delete, rm <project>            Delete a config
  completion <shell>              Generate shell completion (bash|zsh|fish)
  version                         Print version
  help                            Show this help

Config files: ~/.config/frostmux/ (override with $FROSTMUX_CONFIG)
`)
}
