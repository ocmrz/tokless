//go:build !windows

package util

import (
	"os"
	"syscall"
)

func suspendTTY(restore func()) (func(), bool) {
	restore()
	_, _ = os.Stdout.WriteString("\r\n")
	RestoreConsoleCP()
	_ = syscall.Kill(os.Getpid(), syscall.SIGTSTP)
	return rawMode()
}
