package herdr

import "strings"

// RunningCommand reports whether closing this terminal would end more than
// an idle shell. The foreground record comes from the terminal's own machine.
func (t *Terminal) RunningCommand() bool {
	if t == nil || !t.Live() {
		return false
	}
	if fg := t.Foreground; fg != nil {
		if t.Pid != nil && fg.Pid > 0 && fg.Pid != *t.Pid {
			return true
		}
		name := fg.Name
		if len(fg.Argv) > 0 {
			name = fg.Argv[0]
		}
		if name != "" {
			return !inList(shellNames, base(name)) || shellCommand(fg.Argv)
		}
	}
	if len(t.Argv) == 0 {
		return true // A live terminal without process details is not known idle.
	}
	return !inList(shellNames, base(t.Argv[0])) || shellCommand(t.Argv)
}

func shellCommand(argv []string) bool {
	if len(argv) < 2 {
		return false
	}
	for _, arg := range argv[1:] {
		if arg == "--command" || (strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && strings.Contains(arg, "c")) {
			return true
		}
		if !strings.HasPrefix(arg, "-") {
			return true // A shell executing a script also needs confirmation.
		}
	}
	return false
}
