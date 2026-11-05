//go:build !windows

package ui

import "os/exec"

func hideNotificationWindow(cmd *exec.Cmd) {}
