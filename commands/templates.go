// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"fmt"
	"os"

	"github.com/avitacco/jig/internal/scaffold"
	"github.com/avitacco/jig/internal/template"
	"github.com/spf13/cobra"
)

func (a *App) templatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Useful commands for working with templates",
	}
	cmd.AddCommand(a.templatesDumpCmd())
	return cmd
}

func (a *App) templatesDumpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump <destination>",
		Short: "Dump all available templates to a directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			destination := args[0]
			if _, err := os.Stat(destination); err == nil {
				if err := scaffold.BackupDir(destination); err != nil {
					return fmt.Errorf("failed to back up existing directory: %w", err)
				}
				fmt.Printf("backed up existing directory %s\n", destination)
			}
			return template.DumpTemplates(destination)
		},
	}
	return cmd
}
