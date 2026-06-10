// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"github.com/avitacco/jig/internal/bundle"
	"github.com/spf13/cobra"
)

func (a *App) testCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run module tests",
	}
	cmd.AddCommand(a.testUnitCmd())

	return cmd
}

func (a *App) testUnitCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "unit",
		Short:              "Run unit tests via rake spec through bundle",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return bundle.RunBundle(append([]string{"exec", "rake", "spec"}, args...))
		},
	}
}
