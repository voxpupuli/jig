// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/voxpupuli/jig/v2/internal/scaffold"
)

func (a *App) convertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Bring an existing module onto the jig/voxbox toolchain",
		Long: `Brings a PDK-generated (or hand-maintained) module onto the toolchain
jig's other commands expect.

If metadata.json is missing, jig creates it -- pre-filled from a Modulefile
if one is present, otherwise from an interview (honoring --skip-interview
and the -u/-a/-l/-s/-S flags), with the module name derived from the current
directory's name. jig.toml is written too if it's also missing. If
metadata.json exists but isn't valid JSON, jig reports the parse error and
stops rather than guessing. If it parses but fails validation, jig fills in
the keys it has a safe default for (version, dependencies, requirements,
operatingsystem_support, tags) and warns about the rest, without overwriting
any value that's already present.

It then (re)renders Gemfile, Rakefile, and spec/spec_helper.rb from jig's
embedded templates, creating spec/ if it does not exist.

Unlike jig renew, convert always uses jig's embedded templates -- it does
not consult --template-dir, --template-url, or the module's jig.toml, and it
does not require an allowlist.`,
		// A broken or invalid metadata.json is a module state worth a clear
		// message, not a usage mistake.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			forgeUser, _ := cmd.Flags().GetString("forge-user")
			author, _ := cmd.Flags().GetString("author")
			license, _ := cmd.Flags().GetString("license")
			summary, _ := cmd.Flags().GetString("summary")
			source, _ := cmd.Flags().GetString("source")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			skipInterview, _ := cmd.Flags().GetBool("skip-interview")

			opts := scaffold.ConvertOptions{
				TargetDir: cwd,
				ForgeUser: forgeUser,
				Author:    author,
				License:   license,
				Summary:   summary,
				Source:    source,
				DryRun:    dryRun,
				Out:       cmd.OutOrStdout(),
			}

			metadataPath := filepath.Join(cwd, "metadata.json")
			if _, statErr := os.Stat(metadataPath); os.IsNotExist(statErr) {
				// A Modulefile carries this specific module's own history
				// (its original author, license, ...), so it outranks the
				// generic config defaults -- those describe the person
				// running convert, not necessarily the module's author.
				// Explicit flags still win over everything.
				modulefilePath := filepath.Join(cwd, "Modulefile")
				if mf, mfErr := scaffold.ParseModulefile(modulefilePath); mfErr == nil {
					opts.HasModulefile = true
					mfForgeUser, mfName := scaffold.SplitForgeName(mf.Name)
					if opts.ForgeUser == "" {
						opts.ForgeUser = mfForgeUser
					}
					if opts.Name == "" {
						opts.Name = mfName
					}
					if opts.Author == "" {
						opts.Author = mf.Author
					}
					if opts.License == "" {
						opts.License = mf.License
					}
					if opts.Summary == "" {
						opts.Summary = mf.Summary
						if opts.Summary == "" {
							opts.Summary = mf.Description
						}
					}
					if opts.Source == "" {
						opts.Source = mf.Source
					}
					opts.Version = mf.Version
					opts.Dependencies = mf.Dependencies
				}

				if opts.ForgeUser == "" {
					opts.ForgeUser = a.Config.ForgeUsername
				}
				if opts.Author == "" {
					opts.Author = a.Config.Author
				}
				if opts.License == "" {
					opts.License = a.Config.License
				}
				if opts.License == "" {
					opts.License = "Apache-2.0"
				}

				if opts.ForgeUser == "" || opts.Author == "" {
					currentUser, err := user.Current()
					if err != nil {
						return err
					}
					if opts.ForgeUser == "" {
						opts.ForgeUser = currentUser.Username
					}
					if opts.Author == "" {
						opts.Author = currentUser.Name
					}
				}

				if !skipInterview {
					if err := runConvertInterview(&opts); err != nil {
						return err
					}
				}
			}

			if err := scaffold.ConvertModule(opts); err != nil {
				return fmt.Errorf("convert failed: %w", err)
			}

			if dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), "dry run: no files were changed")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "convert successful: Gemfile, Rakefile, spec/spec_helper.rb")
			}
			return nil
		},
	}

	cmd.Flags().StringP("forge-user", "u", "", "Forge username (used only if metadata.json must be created)")
	cmd.Flags().StringP("author", "a", "", "Author name (used only if metadata.json must be created)")
	cmd.Flags().StringP("license", "l", "", "License type (used only if metadata.json must be created)")
	cmd.Flags().StringP("summary", "s", "", "Summary of the module (used only if metadata.json must be created)")
	cmd.Flags().StringP("source", "S", "", "Source URL for the module (used only if metadata.json must be created)")
	cmd.Flags().BoolP("skip-interview", "i", false, "Skip the interview when metadata.json must be created")
	cmd.Flags().Bool("dry-run", false, "Show what would change without writing any files")

	return cmd
}

func runConvertInterview(opts *scaffold.ConvertOptions) error {
	opts.ForgeUser, _ = prompt("Forge username", opts.ForgeUser)
	opts.Author, _ = prompt("Author name", opts.Author)
	opts.License, _ = prompt("License type", opts.License)
	opts.Summary, _ = prompt("Summary of the module", opts.Summary)
	opts.Source, _ = prompt("Source URL for the module", opts.Source)
	return nil
}
