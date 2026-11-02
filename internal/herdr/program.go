package herdr

import (
	"path"
	"strings"
)

// ProgramInfo is what runs in a terminal, as people see it: the name, icon
// and tile of the app's program table (defaults/programs.json).
type ProgramInfo struct {
	Key   string
	Name  string
	Icon  string // a file under programicons/, without the extension
	Tint  bool
	Shell bool
	Agent bool
}

// programEntry is one row of the app's program table.
type programEntry struct {
	Name string
	Icon string
	Also []string
}

var shellNames = []string{"zsh", "bash", "fish", "sh", "dash", "nu", "pwsh", "powershell", "cmd", "elvish", "xonsh", "tcsh", "csh", "ksh", "ion"}

var interpreterNames = []string{"node", "bun", "bunx", "deno", "python", "python3", "npx", "pnpm", "yarn", "uv", "uvx", "ruby", "bundle", "env", "sudo", "nice", "time", "watch", "nodemon"}

var programTable = map[string]programEntry{
	"claude":   {"Claude Code", "claude", []string{"claude-code"}},
	"codex":    {"Codex", "codex", nil},
	"pi":       {"Pi", "pi", nil},
	"opencode": {"OpenCode", "opencode", nil},
	"nvim":     {"Neovim", "nvim", nil},
	"vim":      {"Vim", "vim", []string{"vi", "view"}},
	"git":      {"Git", "git", nil},
	"gh":       {"GitHub", "gh", nil},
	"htop":     {"Activity", "htop", []string{"btop", "top", "btm", "glances"}},
	"node":     {"Node", "node", nil},
	"python":   {"Python", "python", []string{"python3", "ipython", "pip", "pip3"}},
	"go":       {"Go", "go", nil},
	"cargo":    {"Cargo", "cargo", []string{"rustc", "rustup"}},
	"docker":   {"Docker", "docker", nil},
	"ssh":      {"SSH", "ssh", []string{"mosh", "et"}},
	"tmux":     {"tmux", "tmux", []string{"zellij", "screen"}},
	"rg":       {"Search", "rg", []string{"grep", "fzf", "fd", "find"}},
	"less":     {"Pager", "less", []string{"more", "man", "bat"}},
	"uv":       {"uv", "uv", []string{"uvx"}},
	"shell":    {"Shell", "shell", nil},
	"program":  {"Program", "program", nil},
	"agent":    {"Agent", "agent", nil},
}

// base returns a program's name from a path, without .exe.
func base(s string) string {
	s = path.Base(strings.ReplaceAll(s, "\\", "/"))
	s = strings.TrimLeft(s, "-")
	s = strings.TrimSuffix(s, ".exe")
	return s
}

func inList(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// match finds the table entry for a program or an agent's app name.
func match(name string) (string, programEntry, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	if e, ok := programTable[name]; ok {
		return name, e, true
	}
	for key, e := range programTable {
		for _, also := range e.Also {
			if also == name {
				return key, e, true
			}
		}
	}
	return "", programEntry{}, false
}

// Program resolves what a terminal runs, from the agent's own report else the
// program it was started with (programs.rs, resolve).
func Program(t *Terminal) ProgramInfo {
	if t == nil {
		return ProgramInfo{Key: "shell", Name: "shell", Icon: "shell", Tint: true, Shell: true}
	}
	if app := agentApp(t); app != "" {
		if key, e, ok := match(app); ok {
			return ProgramInfo{Key: key, Name: e.Name, Icon: e.Icon, Tint: true, Agent: true}
		}
		return ProgramInfo{Key: "agent", Name: app, Icon: "agent", Tint: true, Agent: true}
	}
	argv := t.Argv
	// The app starts agents in the person's shell; show the agent, not the shell.
	if len(argv) > 4 && inList(shellNames, base(argv[0])) && strings.HasPrefix(argv[1], "-") && strings.HasSuffix(argv[1], "c") {
		argv = argv[4:]
	}
	if len(argv) == 0 {
		return ProgramInfo{Key: "shell", Name: "shell", Icon: "shell", Tint: true, Shell: true}
	}
	first := base(argv[0])
	if inList(interpreterNames, first) {
		for _, arg := range argv[1:] {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			name := base(arg)
			for _, ext := range []string{".js", ".mjs", ".cjs", ".ts", ".py", ".rb"} {
				name = strings.TrimSuffix(name, ext)
			}
			if key, e, ok := match(name); ok && !inList(interpreterNames, key) {
				return ProgramInfo{Key: key, Name: e.Name, Icon: e.Icon, Tint: true}
			}
		}
	}
	if key, e, ok := match(first); ok {
		return ProgramInfo{Key: key, Name: e.Name, Icon: e.Icon, Tint: true}
	}
	if inList(shellNames, first) {
		return ProgramInfo{Key: "shell", Name: first, Icon: "shell", Tint: true, Shell: true}
	}
	return ProgramInfo{Key: "program", Name: first, Icon: "program", Tint: true}
}

// agentApp is the name an agent reported for itself, if it did.
func agentApp(t *Terminal) string {
	if t.Agent == "" || inList(shellNames, t.Agent) {
		return ""
	}
	if _, _, ok := match(t.Agent); ok {
		return t.Agent
	}
	if t.Foreground != nil && t.Foreground.Name != "" {
		if _, _, ok := match(t.Foreground.Name); ok {
			return t.Foreground.Name
		}
	}
	return ""
}
