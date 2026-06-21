// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"github.com/spf13/cobra"
)

func (a *App) updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "update",
		Short:              "Run msync update through bundle to sync module files",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runBundle(append([]string{"exec", "msync", "update"}, args...))
		},
	}
}
