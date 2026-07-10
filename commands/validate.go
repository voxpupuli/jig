// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"github.com/spf13/cobra"
)

// validateTasks maps the selection flags to the rake tasks to run. With no
// flags set every check runs (mirroring `pdk validate`); setting any flag
// narrows the run to just the selected checks.
func validateTasks(syntax, lint, rubocop bool) []string {
	if !syntax && !lint && !rubocop {
		return []string{"validate", "lint", "rubocop"}
	}
	var tasks []string
	if syntax {
		tasks = append(tasks, "validate")
	}
	if lint {
		tasks = append(tasks, "lint")
	}
	if rubocop {
		tasks = append(tasks, "rubocop")
	}
	return tasks
}

func (a *App) validateCmd() *cobra.Command {
	var syntax, lint, rubocop bool

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Run rake validate, lint and rubocop through bundle",
		Long: `Run validation checks against the current module.

By default all checks run: syntax (rake validate), puppet-lint (rake lint)
and rubocop (rake rubocop). Pass one or more selection flags to run only
those checks. Arguments after -- are forwarded to each rake invocation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// One rake invocation per task: the voxbox entrypoint only honours
			// a single task per run (#64), and locally the behaviour is the
			// same. A failing task exits with the child's code, so later tasks
			// only run once the earlier ones pass.
			for _, task := range validateTasks(syntax, lint, rubocop) {
				if err := a.runRake(append([]string{task}, args...)); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&syntax, "syntax", "s", false, "run syntax checks (rake validate)")
	cmd.Flags().BoolVarP(&lint, "lint", "l", false, "run puppet-lint checks (rake lint)")
	cmd.Flags().BoolVarP(&rubocop, "rubocop", "r", false, "run rubocop checks (rake rubocop)")

	return cmd
}
