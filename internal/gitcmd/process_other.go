//go:build !windows

package gitcmd

import "os/exec"

func configureProcess(cmd *exec.Cmd) {}
