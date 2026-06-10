// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"os"

	"github.com/avitacco/jig/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func Execute() error {
	app := NewApp()

	for _, arg := range os.Args {
		if arg == "--debug" || arg == "-d" {
			app.Logger.SetLevel(logrus.DebugLevel)
			break
		}
	}

	rootCmd := &cobra.Command{
		Use:   "jig",
		Short: "A tool for building and publishing Puppet modules",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")

			cfg, err := config.Load(configPath, app.Logger)
			if err != nil {
				return err
			}

			app.Config = cfg
			return nil
		},
	}

	rootCmd.PersistentFlags().String("config", "", "Path to config file")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug output")

	rootCmd.AddCommand(app.newCmd())
	rootCmd.AddCommand(app.templatesCmd())
	rootCmd.AddCommand(app.buildCmd())
	rootCmd.AddCommand(app.releaseCmd())
	rootCmd.AddCommand(app.updateCmd())
	rootCmd.AddCommand(app.validateCmd())
	rootCmd.AddCommand(app.testCmd())

	return rootCmd.Execute()
}
