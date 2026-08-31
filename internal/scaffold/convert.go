// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/voxpupuli/jig/v2/internal/config"
	"github.com/voxpupuli/jig/v2/internal/module"
	"github.com/voxpupuli/jig/v2/internal/template"
)

// ConvertOptions controls ConvertModule. The metadata fields are only used
// when metadata.json is missing from TargetDir, to seed the one jig creates;
// when metadata.json already exists (valid or repairable) they are ignored.
type ConvertOptions struct {
	TargetDir string
	DryRun    bool
	// Out receives progress messages; defaults to os.Stdout.
	Out io.Writer

	ForgeUser    string
	Name         string
	Author       string
	License      string
	Summary      string
	Source       string
	Version      string
	Dependencies []module.Dependency

	// HasModulefile records whether the options above were pre-filled from a
	// Modulefile, so ConvertModule can warn that it's no longer needed.
	HasModulefile bool
}

// ConvertModule brings an existing module onto the toolchain jig's other
// commands expect: it ensures metadata.json exists and is at least minimally
// valid (creating or repairing it as needed), ensures jig.toml exists, and
// (re)writes Gemfile, Rakefile, and spec/spec_helper.rb from jig's embedded
// templates.
func ConvertModule(opts ConvertOptions) error {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}

	metadataPath := filepath.Join(opts.TargetDir, "metadata.json")
	switch _, err := os.Stat(metadataPath); {
	case err == nil:
		if err := repairMetadata(metadataPath, opts.DryRun, out); err != nil {
			return err
		}
	case os.IsNotExist(err):
		if err := createMetadata(opts, metadataPath, out); err != nil {
			return err
		}
	default:
		return fmt.Errorf("failed to stat %s: %w", metadataPath, err)
	}

	filesToUpdate := []struct {
		TemplatePath string
		DestPath     string
	}{
		{TemplatePath: "module/Gemfile", DestPath: "Gemfile"},
		{TemplatePath: "module/Rakefile", DestPath: "Rakefile"},
		{TemplatePath: "module/spec/spec_helper.rb", DestPath: filepath.Join("spec", "spec_helper.rb")},
	}

	renderer := template.NewRenderer()

	for _, f := range filesToUpdate {
		fullDestPath := filepath.Join(opts.TargetDir, f.DestPath)

		content, err := renderer.Render(f.TemplatePath, nil)
		if err != nil {
			return fmt.Errorf("error while parsing template %s: %w", f.TemplatePath, err)
		}

		if opts.DryRun {
			fmt.Fprintf(out, "would write %s\n", fullDestPath)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fullDestPath), 0755); err != nil {
			return fmt.Errorf("error creating directory for %s: %w", fullDestPath, err)
		}

		if err := os.WriteFile(fullDestPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("error writing file %s: %w", fullDestPath, err)
		}
	}

	return nil
}

// createMetadata builds metadata.json from opts (filling in the module name
// from the target directory when it wasn't already resolved from a
// Modulefile) and writes it, along with jig.toml if that's also missing.
func createMetadata(opts ConvertOptions, metadataPath string, out io.Writer) error {
	name := opts.Name
	if name == "" {
		_, name = SplitForgeName(filepath.Base(opts.TargetDir))
	}

	meta := module.NewMetadata(name, opts.ForgeUser, opts.Author)
	if opts.License != "" {
		meta.License = opts.License
	}
	meta.Summary = opts.Summary
	meta.Source = opts.Source
	if opts.Version != "" {
		meta.Version = opts.Version
	}
	if len(opts.Dependencies) > 0 {
		meta.Dependencies = opts.Dependencies
	}

	if opts.DryRun {
		fmt.Fprintf(out, "would create %s\n", metadataPath)
	} else {
		if err := meta.Write(metadataPath); err != nil {
			return fmt.Errorf("failed to write metadata.json: %w", err)
		}
		fmt.Fprintf(out, "created %s\n", metadataPath)
	}

	jigTomlPath := filepath.Join(opts.TargetDir, config.ModuleConfigFileName)
	if _, err := os.Stat(jigTomlPath); os.IsNotExist(err) {
		if opts.DryRun {
			fmt.Fprintf(out, "would create %s\n", jigTomlPath)
		} else {
			if err := (config.ModuleConfig{}).Write(opts.TargetDir); err != nil {
				return err
			}
			fmt.Fprintf(out, "created %s\n", jigTomlPath)
		}
	}

	if opts.HasModulefile {
		fmt.Fprintf(out, "warning: Modulefile is no longer used; its contents were carried into metadata.json. You can delete it.\n")
	}

	return nil
}

// repairMetadata reads an existing metadata.json. A JSON parse error is
// fatal -- jig won't guess at a broken file. Otherwise it fills in the keys
// that have a safe default (version, and the dependencies/requirements/
// operatingsystem_support/tags lists) when they're missing, warns about
// validation failures it can't fix on its own, and writes the file back only
// if something actually changed, so running convert twice is a no-op.
func repairMetadata(metadataPath string, dryRun bool, out io.Writer) error {
	meta, err := module.ReadMetadata(metadataPath)
	if err != nil {
		return fmt.Errorf("%s: %w", metadataPath, err)
	}

	changed := false
	if meta.Version == "" {
		meta.Version = "0.1.0"
		fmt.Fprintf(out, "warning: version missing from metadata.json, defaulting to 0.1.0\n")
		changed = true
	}
	if meta.Dependencies == nil {
		meta.Dependencies = []module.Dependency{}
		changed = true
	}
	if meta.Requirements == nil {
		meta.Requirements = []module.Requirement{}
		changed = true
	}
	if meta.OperatingSystem == nil {
		meta.OperatingSystem = []module.OperatingSystem{}
		changed = true
	}
	if meta.Tags == nil {
		meta.Tags = []string{}
		changed = true
	}

	for _, r := range meta.Validate() {
		fmt.Fprintf(out, "warning: %s - %s\n", r.Field, r.Message)
	}

	if !changed {
		return nil
	}

	if dryRun {
		fmt.Fprintf(out, "would repair %s\n", metadataPath)
		return nil
	}

	if err := meta.Write(metadataPath); err != nil {
		return fmt.Errorf("failed to write repaired metadata.json: %w", err)
	}
	fmt.Fprintf(out, "repaired %s\n", metadataPath)
	return nil
}
