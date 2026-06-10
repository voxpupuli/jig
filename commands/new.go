// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/avitacco/jig/internal/scaffold"
	"github.com/spf13/cobra"
)

func (a *App) newCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create new things",
	}
	cmd.PersistentFlags().StringP("template-dir", "t", "", "Path to custom template directory")
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			forgeUser, _ := cmd.Flags().GetString("forge-user")
			license, _ := cmd.Flags().GetString("license")
			summary, _ := cmd.Flags().GetString("summary")
			source, _ := cmd.Flags().GetString("source")
			author, _ := cmd.Flags().GetString("author")
			force, _ := cmd.Flags().GetBool("force")
			templateDir, _ := cmd.InheritedFlags().GetString("template-dir")

			if templateDir == "" {
				templateDir = a.Config.TemplateDir
			}

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

			opts := scaffold.Options{
				ForgeUser:   forgeUser,
				Name:        args[0],
				License:     license,
				Summary:     summary,
				Source:      source,
				Author:      author,
				Force:       force,
				TemplateDir: templateDir,
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

func (a *App) newClassCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "class <name>",
		Short: "Create a new Puppet class",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateDir, _ := cmd.InheritedFlags().GetString("template-dir")

			if templateDir == "" {
				templateDir = a.Config.TemplateDir
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			opts := scaffold.ComponentOptions{
				Name:        args[0],
				TemplateDir: templateDir,
				WorkDir:     cwd,
			}
			return scaffold.NewClass(opts)
		},
	}
	return cmd
}

func (a *App) newDefinedTypeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "defined_type <name>",
		Short: "Create a new Puppet defined type",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateDir, _ := cmd.InheritedFlags().GetString("template-dir")

			if templateDir == "" {
				templateDir = a.Config.TemplateDir
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			opts := scaffold.ComponentOptions{
				Name:        args[0],
				TemplateDir: templateDir,
				WorkDir:     cwd,
			}
			return scaffold.NewDefinedType(opts)
		},
	}
	return cmd
}

func (a *App) newFactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fact <name>",
		Short: "Create a new Puppet fact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateDir, _ := cmd.InheritedFlags().GetString("template-dir")

			if templateDir == "" {
				templateDir = a.Config.TemplateDir
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			opts := scaffold.ComponentOptions{
				Name:        args[0],
				TemplateDir: templateDir,
				WorkDir:     cwd,
			}
			return scaffold.NewFact(opts)
		},
	}
	return cmd
}

func (a *App) newFunctionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "function <name>",
		Short: "Create a new Puppet function",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateDir, _ := cmd.InheritedFlags().GetString("template-dir")

			if templateDir == "" {
				templateDir = a.Config.TemplateDir
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			opts := scaffold.ComponentOptions{
				Name:        args[0],
				TemplateDir: templateDir,
				WorkDir:     cwd,
			}
			return scaffold.NewFunction(opts)
		},
	}
	return cmd
}

func (a *App) newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task <name>",
		Short: "Create a new Puppet task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateDir, _ := cmd.InheritedFlags().GetString("template-dir")

			if templateDir == "" {
				templateDir = a.Config.TemplateDir
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			opts := scaffold.ComponentOptions{
				Name:        args[0],
				TemplateDir: templateDir,
				WorkDir:     cwd,
			}
			return scaffold.NewTask(opts)
		},
	}
	return cmd
}

func (a *App) newProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider <name>",
		Short: "Create a new Puppet provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateDir, _ := cmd.InheritedFlags().GetString("template-dir")

			if templateDir == "" {
				templateDir = a.Config.TemplateDir
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			opts := scaffold.ComponentOptions{
				Name:        args[0],
				TemplateDir: templateDir,
				WorkDir:     cwd,
			}
			return scaffold.NewProvider(opts)
		},
	}
	return cmd
}

func (a *App) newTransportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transport <name>",
		Short: "Create a new Puppet transport",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateDir, _ := cmd.InheritedFlags().GetString("template-dir")

			if templateDir == "" {
				templateDir = a.Config.TemplateDir
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			opts := scaffold.ComponentOptions{
				Name:        args[0],
				TemplateDir: templateDir,
				WorkDir:     cwd,
			}
			return scaffold.NewTransport(opts)
		},
	}
	return cmd
}

func (a *App) newTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test <n>",
		Short: "Create a new unit test for an existing class or defined type",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateDir, _ := cmd.InheritedFlags().GetString("template-dir")

			if templateDir == "" {
				templateDir = a.Config.TemplateDir
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			opts := scaffold.ComponentOptions{
				Name:        args[0],
				TemplateDir: templateDir,
				WorkDir:     cwd,
			}
			return scaffold.NewTest(opts)
		},
	}
	return cmd
}
