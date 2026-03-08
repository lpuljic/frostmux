package main

import (
	"flag"
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

	// bare `frostmux myproject` falls through to default and just starts it,
	// so you don't have to type `frostmux start myproject` every time
	var err error
	switch cmd {
	case "start", "s":
		err = cmdStart(args)
	case "stop", "st":
		err = cmdStop(args)
	case "list", "ls":
		err = cmdList()
	case "freeze":
		err = cmdFreeze(args)
	case "init":
		err = cmdInit(args)
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
		err = cmdStart([]string{cmd})
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "frostmux: %v\n", err)
		os.Exit(1)
	}
}

func cmdStart(args []string) error {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	file := fs.String("f", "", "path to config file")
	fs.Parse(args)

	var cfgPath string
	var err error

	if *file != "" {
		cfgPath = *file
	} else if fs.NArg() > 0 {
		cfgPath, err = config.FindConfig(fs.Arg(0))
	} else {
		cfgPath, err = config.FindConfig("")
	}
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
	configs, err := session.ListConfigs()
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

// cmdFreeze snapshots a live tmux session into YAML.
// With a name it saves to disk, without it dumps to stdout so you can pipe it.
func cmdFreeze(args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	mgr := session.NewManager(tmux.NewClient())
	cfg, err := mgr.Freeze(name)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if name != "" {
		dir := config.Dir()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating config dir: %w", err)
		}

		path := filepath.Join(dir, name+".yml")
		_, exists := os.Stat(path)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
		if exists == nil {
			fmt.Printf("updated %s\n", path)
		} else {
			fmt.Printf("created %s\n", path)
		}
	} else {
		fmt.Print(string(data))
	}
	return nil
}

// cmdInit looks at the project dir (go.mod, package.json, etc.) and
// generates a reasonable starting config. Opens $EDITOR so you can tweak it.
func cmdInit(args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	cfg := config.Detect(absDir)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	cfgDir := config.Dir()
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	path := filepath.Join(cfgDir, cfg.Session+".yml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config %q already exists, use 'frostmux edit %s' to modify", path, cfg.Session)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("created %s\n", path)
	return openEditor(path)
}

func cmdNew(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: frostmux new <project>")
	}

	name := args[0]
	dir := config.Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	path := filepath.Join(dir, name+".yml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config %q already exists", name)
	}

	scaffold := config.Config{
		Session: name,

		Windows: []config.Window{
			{Name: "editor", Panes: []config.Pane{{Command: "$EDITOR ."}}},
			{Name: "shell", Panes: []config.Pane{{}}},
		},
	}

	data, err := yaml.Marshal(&scaffold)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}

	fmt.Printf("created %s\n", path)
	return openEditor(path)
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
    commands="start stop list freeze init new edit delete completion help version"

    case "$prev" in
        start|s|stop|st|edit|e|delete|rm)
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
        'start:Start a session from config'
        's:Start a session from config'
        'stop:Kill a tmux session'
        'st:Kill a tmux session'
        'list:List available configs'
        'ls:List available configs'
        'freeze:Capture current session to YAML'
        'init:Detect project type and scaffold config'
        'new:Create a new config'
        'n:Create a new config'
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
                start|s|stop|st|edit|e|delete|rm)
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
complete -c frostmux -n '__fish_use_subcommand' -a 'start stop list freeze init new edit delete completion help version'
complete -c frostmux -n '__fish_use_subcommand' -a '(frostmux list 2>/dev/null)'
complete -c frostmux -n '__fish_seen_subcommand_from start s stop st edit e delete rm' -a '(frostmux list 2>/dev/null)'
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

Commands:
  start, s   <project> [-f file]  Start a session from config
  stop, st   <project>            Kill a tmux session
  list, ls                        List available configs
  freeze     [name]               Capture current session to YAML
  init       [dir]                Detect project type, scaffold config
  new, n     <project>            Create a new config in $EDITOR
  edit, e    <project>            Edit an existing config
  delete, rm <project>            Delete a config
  completion <shell>              Generate shell completion (bash|zsh|fish)
  version                         Print version
  help                            Show this help

Shortcut:
  frostmux <project>                Same as 'frostmux start <project>'

Config files: ~/.config/frostmux/ (override with $frostmux_CONFIG)
`)
}
