// Package gitcmd runs the desktop app's local Git commands.
package gitcmd

import "os/exec"

// Output runs Git in dir and returns its stdout.
func Output(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	configureProcess(cmd)
	return cmd.Output()
}
