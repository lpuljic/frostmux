package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lpuljic/muxify/internal/config"
	"github.com/lpuljic/muxify/internal/session"
	"github.com/lpuljic/muxify/internal/tmux"
	"gopkg.in/yaml.v3"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

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
		fmt.Printf("muxify %s\n", version)
		return
	default:
		err = cmdStart([]string{cmd})
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "muxify: %v\n", err)
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
		return fmt.Errorf("usage: muxify stop <project>")
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
		return fmt.Errorf("config %q already exists, use 'muxify edit %s' to modify", path, cfg.Session)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("created %s\n", path)
	return openEditor(path)
}

func cmdNew(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: muxify new <project>")
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
		return fmt.Errorf("usage: muxify edit <project>")
	}

	path, err := config.FindConfig(args[0])
	if err != nil {
		return err
	}

	return openEditor(path)
}

func cmdDelete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: muxify delete <project>")
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
		return fmt.Errorf("usage: muxify completion <bash|zsh|fish>")
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

const bashCompletion = `_muxify() {
    local cur prev commands
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    commands="start stop list freeze init new edit delete completion help version"

    case "$prev" in
        start|s|stop|st|edit|e|delete|rm)
            COMPREPLY=($(compgen -W "$(muxify list 2>/dev/null)" -- "$cur"))
            return
            ;;
        muxify)
            COMPREPLY=($(compgen -W "$commands $(muxify list 2>/dev/null)" -- "$cur"))
            return
            ;;
        completion)
            COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur"))
            return
            ;;
    esac
}
complete -F _muxify muxify
`

const zshCompletion = `_muxify() {
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
            projects=(${(f)"$(muxify list 2>/dev/null)"})
            _describe 'command' commands
            [[ ${#projects} -gt 0 ]] && _describe 'project' projects
            ;;
        args)
            case "$words[2]" in
                start|s|stop|st|edit|e|delete|rm)
                    projects=(${(f)"$(muxify list 2>/dev/null)"})
                    [[ ${#projects} -gt 0 ]] && _describe 'project' projects
                    ;;
                completion)
                    _describe 'shell' '(bash zsh fish)'
                    ;;
            esac
            ;;
    esac
}

compdef _muxify muxify
`

const fishCompletion = `complete -c muxify -f
complete -c muxify -n '__fish_use_subcommand' -a 'start stop list freeze init new edit delete completion help version'
complete -c muxify -n '__fish_use_subcommand' -a '(muxify list 2>/dev/null)'
complete -c muxify -n '__fish_seen_subcommand_from start s stop st edit e delete rm' -a '(muxify list 2>/dev/null)'
complete -c muxify -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'
`

func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func usage() {
	fmt.Fprintf(os.Stderr, `muxify — tmux session manager

Capture first, config second.

Usage:
  muxify <command> [arguments]

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
  muxify <project>                Same as 'muxify start <project>'

Config files: ~/.config/muxify/ (override with $MUXIFY_CONFIG)
`)
}
