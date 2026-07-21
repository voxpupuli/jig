// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/spf13/cobra"
	"github.com/voxpupuli/jig/internal/scaffold"
)

func (a *App) newCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create new things",
	}
	addTemplateSourceFlags(cmd)
	cmd.AddCommand(a.newModuleCmd())
	cmd.AddCommand(a.newClassCmd())
	cmd.AddCommand(a.newDefinedTypeCmd())
	cmd.AddCommand(a.newFactCmd())
	cmd.AddCommand(a.newFunctionCmd())
	cmd.AddCommand(a.newTaskCmd())
	cmd.AddCommand(a.newProviderCmd())
	cmd.AddCommand(a.newTransportCmd())
	cmd.AddCommand(a.newTestCmd())
	return cmd
}

func (a *App) newModuleCmd() *cobra.Command {
	newModuleCmd := &cobra.Command{
		Use:   "module <name>",
		Short: "Create a new Puppet module",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			forgeUser, _ := cmd.Flags().GetString("forge-user")
			license, _ := cmd.Flags().GetString("license")
			summary, _ := cmd.Flags().GetString("summary")
			source, _ := cmd.Flags().GetString("source")
			author, _ := cmd.Flags().GetString("author")
			force, _ := cmd.Flags().GetBool("force")

			if forgeUser == "" {
				forgeUser = a.Config.ForgeUsername
			}

			if author == "" {
				author = a.Config.Author
			}

			if forgeUser == "" || author == "" {
				currentUser, err := user.Current()
				if err != nil {
					return err
				}
				if forgeUser == "" {
					forgeUser = currentUser.Username
				}
				if author == "" {
					author = currentUser.Name
				}
			}

			if license == "" {
				license = a.Config.License
			}

			if license == "" {
				license = "Apache-2.0"
			}

			// No metadata.json exists yet, so only flags and config feed
			// the template source here.
			src, err := a.resolveTemplateSource(cmd.InheritedFlags(), "")
			if err != nil {
				return err
			}
			defer src.Cleanup()

			opts := scaffold.Options{
				ForgeUser:      forgeUser,
				Name:           args[0],
				License:        license,
				Summary:        summary,
				Source:         source,
				Author:         author,
				Force:          force,
				TemplateDir:    src.Dir,
				TemplateURL:    src.URL,
				TemplateRef:    src.Ref,
				TemplateCommit: src.Commit,
			}

			skipInterview, _ := cmd.Flags().GetBool("skip-interview")
			if !skipInterview {
				err := runModuleInterview(&opts)
				if err != nil {
					return err
				}
			}

			return scaffold.NewModule(opts)
		},
	}

	newModuleCmd.Flags().StringP("forge-user", "u", "", "Forge username")
	newModuleCmd.Flags().StringP("author", "a", "", "Author name")
	newModuleCmd.Flags().StringP("license", "l", "", "License type")
	newModuleCmd.Flags().StringP("summary", "s", "", "Summary of the module")
	newModuleCmd.Flags().StringP("source", "S", "", "Source URL for the module")
	newModuleCmd.Flags().BoolP("force", "f", false, "Force creation of the module even if it already exists. Note: a backup of the existing directory will be created.")
	newModuleCmd.Flags().BoolP("skip-interview", "i", false, "Skip interview questions")

	return newModuleCmd
}

func runModuleInterview(opts *scaffold.Options) error {
	opts.ForgeUser, _ = prompt("Forge username", opts.ForgeUser)
	opts.Author, _ = prompt("Author name", opts.Author)
	opts.License, _ = prompt("License type", opts.License)
	opts.Summary, _ = prompt("Summary of the module", opts.Summary)
	opts.Source, _ = prompt("Source URL for the module", opts.Source)
	return nil
}

func prompt(question string, defaultVal string) (string, error) {
	fmt.Printf("%s [%s]: ", question, defaultVal)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return defaultVal, nil
	}
	return input, nil
}

// componentRunE builds the RunE shared by all component subcommands: resolve
// the template source (flags, then the module's recorded template-url, then
// config), scaffold in the current working directory, and clean up any
// temporary template clone.
func (a *App) componentRunE(newFn func(scaffold.ComponentOptions) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		src, err := a.resolveTemplateSource(cmd.InheritedFlags(), cwd)
		if err != nil {
			return err
		}
		defer src.Cleanup()

		opts := scaffold.ComponentOptions{
			Name:        args[0],
			TemplateDir: src.Dir,
			WorkDir:     cwd,
		}
		return newFn(opts)
	}
}

func (a *App) newClassCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "class <name>",
		Short: "Create a new Puppet class",
		Args:  cobra.ExactArgs(1),
		RunE:  a.componentRunE(scaffold.NewClass),
	}
}

func (a *App) newDefinedTypeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "defined_type <name>",
		Short: "Create a new Puppet defined type",
		Args:  cobra.ExactArgs(1),
		RunE:  a.componentRunE(scaffold.NewDefinedType),
	}
}

func (a *App) newFactCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fact <name>",
		Short: "Create a new Puppet fact",
		Args:  cobra.ExactArgs(1),
		RunE:  a.componentRunE(scaffold.NewFact),
	}
}

func (a *App) newFunctionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "function <name>",
		Short: "Create a new Puppet function",
		Args:  cobra.ExactArgs(1),
		RunE:  a.componentRunE(scaffold.NewFunction),
	}
}

func (a *App) newTaskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "task <name>",
		Short: "Create a new Puppet task",
		Args:  cobra.ExactArgs(1),
		RunE:  a.componentRunE(scaffold.NewTask),
	}
}

func (a *App) newProviderCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "provider <name>",
		Short: "Create a new Puppet provider",
		Args:  cobra.ExactArgs(1),
		RunE:  a.componentRunE(scaffold.NewProvider),
	}
}

func (a *App) newTransportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "transport <name>",
		Short: "Create a new Puppet transport",
		Args:  cobra.ExactArgs(1),
		RunE:  a.componentRunE(scaffold.NewTransport),
	}
}

func (a *App) newTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test <n>",
		Short: "Create a new unit test for an existing class or defined type",
		Args:  cobra.ExactArgs(1),
		RunE:  a.componentRunE(scaffold.NewTest),
	}
}
