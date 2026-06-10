// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"github.com/avitacco/jig/internal/bundle"
	"github.com/spf13/cobra"
)

func (a *App) updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "update",
		Short:              "Run msync update through bundle to sync module files",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return bundle.RunBundle(append([]string{"exec", "msync", "update"}, args...))
		},
	}
}
