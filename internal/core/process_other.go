//go:build !windows

package core

import "os/exec"

func configureHCProcess(cmd *exec.Cmd) {}
