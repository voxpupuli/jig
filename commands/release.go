// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"fmt"
	"os"

	"github.com/avitacco/jig/internal/forge"
	"github.com/avitacco/jig/internal/release"
	"github.com/spf13/cobra"
)

func (a *App) releaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Build and publish a Puppet module release to the Forge",
		RunE: func(cmd *cobra.Command, args []string) error {
			version, _ := cmd.Flags().GetString("version")
			token, _ := cmd.Flags().GetString("token")
			skipValidation, _ := cmd.Flags().GetBool("skip-validation")
			skipBuild, _ := cmd.Flags().GetBool("skip-build")
			skipPublish, _ := cmd.Flags().GetBool("skip-publish")

			if token == "" {
				token = a.Config.ForgeToken
			}

			if token == "" && !skipPublish {
				return fmt.Errorf("a Forge API token is required; set forge_token in your config or pass --token")
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			opts := release.Options{
				Version:        version,
				SkipValidation: skipValidation,
				SkipBuild:      skipBuild,
				SkipPublish:    skipPublish,
			}

			publisher := forge.NewPublisher(token)

			return release.DoRelease(cwd, opts, publisher)
		},
	}

	cmd.Flags().StringP("version", "v", "", "Version to release (required, e.g. 1.2.3)")
	cmd.Flags().StringP("token", "k", "", "Forge API token (overrides config)")
	cmd.Flags().Bool("skip-validation", false, "Skip metadata validation")
	cmd.Flags().Bool("skip-build", false, "Skip building the module archive")
	cmd.Flags().Bool("skip-publish", false, "Skip publishing to the Forge")

	_ = cmd.MarkFlagRequired("version")

	return cmd
}
