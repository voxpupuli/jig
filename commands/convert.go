package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/voxpupuli/jig/internal/scaffold"
)

func (a *App) convertCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "convert",
		Short: "Update Gemfile, Rakefile and spec_helper.rb in an existing module to be usable with voxbox and Vox Pupuli tooling",
		Long:  `Reads the templates and overwrites Gemfile, Rakefile uand spec/spec_helper.rb in the actual Puppet module.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			err = scaffold.ConvertModule(cwd)
			if err != nil {
				return fmt.Errorf("convert failed: %w", err)
			}

			fmt.Println("convert successful: Gemfile, Rakefile, spec/spec_helper.rb")
			return nil
		},
	}
}

