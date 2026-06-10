// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"fmt"
	"os"

	"github.com/avitacco/jig/internal/build"
	"github.com/spf13/cobra"
)

func (a *App) buildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a Puppet module package",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
			return build.DoBuild(cwd)
		},
	}
	return cmd
}
