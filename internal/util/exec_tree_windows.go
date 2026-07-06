//go:build windows

package util

import (
	"os/exec"
	"strconv"
)

func prepareCommandForTreeKill(_ *exec.Cmd) {}

func killProcessTree(c *exec.Cmd) {
	if c.Process == nil {
		return
	}
	_ = exec.Command("taskkill", "/PID", strconv.Itoa(c.Process.Pid), "/T", "/F").Run()
	_ = c.Process.Kill()
}
