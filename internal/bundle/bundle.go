// SPDX-License-Identifier: GPL-3.0-or-later
package bundle

import (
	"errors"
	"os"
	"os/exec"
)

// indirection seams for testing
var (
	execCommand = exec.Command
	osExit      = os.Exit
)

func RunBundle(args []string) error {
	cmd := execCommand("bundle", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			osExit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}
