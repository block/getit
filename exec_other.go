//go:build !unix

package getit

import (
	"os/exec"
	"time"
)

func killProcessGroup(cmd *exec.Cmd) {
	cmd.WaitDelay = 10 * time.Second
}
