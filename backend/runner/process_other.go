//go:build !linux && !darwin

package runner

import (
	"os/exec"
	"time"
)

func configureCancellation(cmd *exec.Cmd) { cmd.WaitDelay = 2 * time.Second }
