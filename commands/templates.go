// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/voxpupuli/jig/internal/scaffold"
	"github.com/voxpupuli/jig/internal/template"
)

func (a *App) templatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Useful commands for working with templates",
	}
	addTemplateSourceFlags(cmd)
	cmd.AddCommand(a.templatesDumpCmd())
	cmd.AddCommand(a.templatesResolveCmd())
	return cmd
}

func (a *App) templatesResolveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve <name>",
		Short: "Show where a template name resolves from",
		Long: `Trace how a logical template name (e.g. class/class.pp) resolves: which
template source is in effect, every path checked in order, and the winning
file. Uses the same lookup as the commands that render templates, so the
result is exactly what "jig new" would use.`,
		Args: cobra.ExactArgs(1),
		// A not-found template is a legitimate diagnostic result, not a usage
		// mistake; keep the trace readable.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			src, err := a.resolveTemplateSource(cmd, cwd)
			if err != nil {
				return err
			}
			defer src.Cleanup()

			return explainTemplate(cmd.OutOrStdout(), src, args[0])
		},
	}
	return cmd
}

// explainTemplate prints the resolution trace for one logical template name
// against the given template source.
func explainTemplate(w io.Writer, src *templateSource, name string) error {
	switch {
	case src.URL != "":
		fmt.Fprintf(w, "external templates: %s (from %s)\n", src.URL, src.Origin)
		if src.Ref != "" {
			fmt.Fprintf(w, "  ref: %s\n", src.Ref)
		}
		fmt.Fprintf(w, "  commit: %s\n", src.Commit)
		fmt.Fprintf(w, "  cloned to: %s\n", src.Dir)
	case src.Dir != "":
		fmt.Fprintf(w, "external template directory: %s (from %s)\n", src.Dir, src.Origin)
	default:
		fmt.Fprintln(w, "no external template directory configured; using embedded templates only")
	}

	res, err := template.NewRendererWithExternalDir(src.Dir).Explain(name)
	if err != nil {
		return err
	}

	lastSource := ""
	for _, step := range res.Steps {
		if step.Source == template.SourceEmbedded && lastSource == template.SourceExternal {
			fmt.Fprintln(w, "not found in external template directory, falling back to embedded templates")
		}
		state := "not found"
		if step.Found {
			state = "found"
		}
		fmt.Fprintf(w, "  looking for %s (%s) ... %s\n", step.Path, step.Source, state)
		lastSource = step.Source
	}

	if !res.Found {
		return fmt.Errorf("template %s not found in any source", res.Name)
	}

	how := "copied verbatim"
	if res.IsTemplate {
		how = "rendered with text/template"
	}
	fmt.Fprintf(w, "resolved %s to %s template %s (%s)\n", res.Name, res.Source, res.Path, how)
	return nil
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
