// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"github.com/spf13/cobra"
)

func (a *App) msyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "msync [command]",
		Short:              "Run msync through bundle (e.g. jig msync update)",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runBundle(append([]string{"exec", "msync"}, args...))
		},
	}
}
