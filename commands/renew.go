// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/voxpupuli/jig/internal/config"
	"github.com/voxpupuli/jig/internal/scaffold"
)

func (a *App) renewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "renew",
		Short: "Re-render allowlisted module files from the latest templates",
		Long: `Re-render the module's template-managed files and overwrite them with the
latest template output, so template changes can be rolled out across many
modules without hand-editing each one.

Only files matching the [renew] paths allowlist in jig.toml are touched.
The allowlist is empty by default, so nothing is overwritten until the
module opts in; patterns are gitignore-style globs matched against paths
relative to the module root.

The template source is resolved like every other command: the
--template-url/--template-dir flags first, then the [template] section of
jig.toml (re-fetching the latest commit of the recorded ref), then
template_dir from the jig config. After a successful renew from a remote
repository, the [template] section of jig.toml is updated to the commit
that was fetched.`,
		Args: cobra.NoArgs,
		// An empty or unmatched allowlist is a configuration state worth a
		// clear message, not a usage mistake.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			moduleConfig, err := config.LoadModuleConfig(cwd)
			if err != nil {
				return err
			}
			if len(moduleConfig.Renew.Paths) == 0 {
				return fmt.Errorf("nothing to renew: the [renew] paths allowlist in %s is empty; list the files jig renew may re-render and overwrite there", config.ModuleConfigFileName)
			}

			src, err := a.resolveTemplateSource(cmd.Flags(), cwd)
			if err != nil {
				return err
			}
			defer src.Cleanup()

			dryRun, _ := cmd.Flags().GetBool("dry-run")
			if err := scaffold.Renew(scaffold.RenewOptions{
				ModuleDir:   cwd,
				TemplateDir: src.Dir,
				Paths:       moduleConfig.Renew.Paths,
				DryRun:      dryRun,
				Out:         cmd.OutOrStdout(),
			}); err != nil {
				return err
			}

			// Record the source the module was just renewed from, so the next
			// renew starts at this commit's provenance. Rewriting jig.toml
			// regenerates it, so only do it when something actually changed.
			if !dryRun && src.URL != "" {
				renewedFrom := config.ModuleTemplate{URL: src.URL, Ref: src.Ref, Commit: src.Commit}
				if moduleConfig.Template != renewedFrom {
					moduleConfig.Template = renewedFrom
					if err := moduleConfig.Write(cwd); err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "updated [template] in %s to commit %s\n", config.ModuleConfigFileName, src.Commit)
				}
			}
			return nil
		},
	}

	addTemplateSourceFlags(cmd)
	cmd.Flags().Bool("dry-run", false, "Show a diff of what would change without writing any files")
	return cmd
}
