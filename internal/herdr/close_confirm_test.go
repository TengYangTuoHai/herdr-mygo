package herdr

import "testing"

func TestTerminalRunningCommand(t *testing.T) {
	pid := 10
	cases := []struct {
		name string
		term *Terminal
		want bool
	}{
		{"missing", nil, false},
		{"finished", &Terminal{State: "finished", Argv: []string{"node"}}, false},
		{"idle shell", &Terminal{State: "live", Argv: []string{"/bin/zsh", "-l"}}, false},
		{"foreground shell", &Terminal{State: "live", Pid: &pid, Argv: []string{"zsh"}, Foreground: &Foreground{Pid: 10, Name: "zsh"}}, false},
		{"stale agent report", &Terminal{State: "live", Argv: []string{"zsh"}, Status: &Status{App: "claude", State: "done"}, Foreground: &Foreground{Name: "zsh"}}, false},
		{"foreground command", &Terminal{State: "live", Argv: []string{"zsh"}, Foreground: &Foreground{Name: "sleep", Argv: []string{"sleep", "60"}}}, true},
		{"nested shell", &Terminal{State: "live", Pid: &pid, Argv: []string{"zsh"}, Foreground: &Foreground{Pid: 11, Name: "bash"}}, true},
		{"direct program", &Terminal{State: "live", Argv: []string{"python", "server.py"}}, true},
		{"shell command", &Terminal{State: "live", Argv: []string{"zsh", "-lc", "sleep 60"}}, true},
		{"shell script", &Terminal{State: "live", Argv: []string{"bash", "script.sh"}}, true},
		{"unknown live", &Terminal{State: "live"}, true},
		{"finished foreground", &Terminal{State: "finished", Foreground: &Foreground{Name: "claude"}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.term.RunningCommand(); got != tc.want {
				t.Fatalf("RunningCommand() = %v, want %v", got, tc.want)
			}
		})
	}
}
