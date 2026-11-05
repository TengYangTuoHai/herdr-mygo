package core

import (
	"os/exec"
	"syscall"
)

func configureHCProcess(cmd *exec.Cmd) {
	// hc connect communicates through pipes; it must not open a console when
	// started by the desktop app. Keep the core's own daemon lifecycle intact.
	const createNoWindow = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
