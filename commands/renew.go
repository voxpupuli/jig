// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/voxpupuli/jig/v2/internal/config"
	"github.com/voxpupuli/jig/v2/internal/scaffold"
	"github.com/voxpupuli/jig/v2/internal/template"
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
that was fetched.

Templates render with the module's [template.vars] from jig.toml. A
--template-var flag overrides a value and, unless --dry-run is given, is
saved to [template.vars] so later renews keep it. Allowlisted files the
template no longer generates (a [[files]] when rule turned false) are
reported, never deleted.`,
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

			manifest, flagVars, err := loadTemplateVars(cmd.Flags(), src)
			if err != nil {
				return err
			}
			vars, err := manifest.ResolveVars(flagVars, template.VarLayers{Module: moduleConfig.Template.Vars})
			if err != nil {
				return err
			}

			dryRun, _ := cmd.Flags().GetBool("dry-run")
			if err := scaffold.Renew(scaffold.RenewOptions{
				ModuleDir:   cwd,
				TemplateDir: src.Dir,
				Paths:       moduleConfig.Renew.Paths,
				DryRun:      dryRun,
				Out:         cmd.OutOrStdout(),
				Manifest:    manifest,
				Vars:        vars,
			}); err != nil {
				return err
			}
			if dryRun {
				return nil
			}

			// Record the source the module was just renewed from, so the next
			// renew starts at this commit's provenance, and any --template-var
			// values, so later renews keep them. Rewriting jig.toml
			// regenerates it, so only do it when something actually changed.
			updated := moduleConfig.Template
			if src.URL != "" {
				updated.URL, updated.Ref, updated.Commit = src.URL, src.Ref, src.Commit
			}
			if len(flagVars) > 0 {
				updated.Vars = maps.Clone(moduleConfig.Template.Vars)
				if updated.Vars == nil {
					updated.Vars = map[string]any{}
				}
				maps.Copy(updated.Vars, flagVars)
			}
			sourceChanged := !updated.SameSource(moduleConfig.Template)
			varsChanged := !config.VarsEqual(updated.Vars, moduleConfig.Template.Vars)
			if !sourceChanged && !varsChanged {
				return nil
			}

			moduleConfig.Template = updated
			if err := moduleConfig.Write(cwd); err != nil {
				return err
			}
			if sourceChanged {
				fmt.Fprintf(cmd.OutOrStdout(), "updated [template] in %s to commit %s\n", config.ModuleConfigFileName, src.Commit)
			}
			if varsChanged {
				fmt.Fprintf(cmd.OutOrStdout(), "updated [template.vars] in %s: %s\n", config.ModuleConfigFileName, strings.Join(slices.Sorted(maps.Keys(flagVars)), ", "))
			}
			return nil
		},
	}

	addTemplateSourceFlags(cmd)
	cmd.Flags().Bool("dry-run", false, "Show a diff of what would change without writing any files")
	return cmd
}
