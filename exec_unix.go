//go:build unix

package getit

import (
	"os/exec"
	"syscall"
	"time"
)

// killProcessGroup makes context cancellation kill the command's whole
// process group, not just the direct child. Without it, grandchildren (e.g.
// git-remote-https) survive the kill and hold the output pipe open, blocking
// Wait indefinitely. WaitDelay bounds Wait if anything still escapes.
func killProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 10 * time.Second
}
