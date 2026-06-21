// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"github.com/spf13/cobra"
)

func (a *App) validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "validate",
		Short:              "Run rake validate and lint through bundle",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runRake(append([]string{"validate", "lint"}, args...))
		},
	}
}
