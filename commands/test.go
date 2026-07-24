// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
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
	var parallel bool

	cmd := &cobra.Command{
		Use:   "unit",
		Short: "Run unit tests via rake spec through bundle",
		Long: `Run the module's unit tests.

By default this runs serially (rake spec). Pass --parallel to run via
rake parallel_spec instead. Arguments after -- are forwarded to the
rake invocation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			task := "spec"
			if parallel {
				task = "parallel_spec"
			}
			return a.runRake(append([]string{task}, args...))
		},
	}

	cmd.Flags().BoolVar(&parallel, "parallel", false, "run tests in parallel (rake parallel_spec)")

	return cmd
}
