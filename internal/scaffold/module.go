// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/avitacco/jig/internal/module"
)

func NewModule(opts Options) error {
	if err := validateComponentName(opts.Name); err != nil {
		return fmt.Errorf("invalid module name: %w", err)
	}

	baseDir := opts.TargetDir
	if baseDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		baseDir = cwd
	}

	moduleDir := filepath.Join(baseDir, opts.Name)

	if _, err := os.Stat(moduleDir); err == nil {
		if !opts.Force {
			return fmt.Errorf("directory %s already exists, use --force to replace it", moduleDir)
		}
		if err := BackupDir(moduleDir); err != nil {
			return fmt.Errorf("failed to back up existing directory: %w", err)
		}
		fmt.Printf("Renamed existing directory %s\n", moduleDir)
	}

	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", moduleDir, err)
	}

	meta := module.NewMetadata(opts.Name, opts.ForgeUser, opts.Author)
	meta.License = opts.License
	meta.Summary = opts.Summary
	meta.Source = opts.Source

	if err := meta.Write(filepath.Join(moduleDir, "metadata.json")); err != nil {
		return fmt.Errorf("failed to write metadata.json: %w", err)
	}

	renderer := newRenderer(opts.TemplateDir)

	templates := []TemplateFile{
		{FileName: "module/manifests/init.pp", Destination: filepath.Join(moduleDir, "manifests", "init.pp")},
		{FileName: "module/README.md", Destination: filepath.Join(moduleDir, "README.md")},
		{FileName: "module/CHANGELOG.md", Destination: filepath.Join(moduleDir, "CHANGELOG.md")},
		{FileName: "module/spec/init_spec.rb", Destination: filepath.Join(moduleDir, "spec", "classes", "init_spec.rb")},
		{FileName: "module/Gemfile", Destination: filepath.Join(moduleDir, "Gemfile")},
		{FileName: "module/Rakefile", Destination: filepath.Join(moduleDir, "Rakefile")},
		{FileName: "module/editorconfig", Destination: filepath.Join(moduleDir, ".editorconfig")},
		{FileName: "module/gitignore", Destination: filepath.Join(moduleDir, ".gitignore")},
		{FileName: "module/overcommit.yml", Destination: filepath.Join(moduleDir, ".overcommit.yml")},
		{FileName: "module/pdkignore", Destination: filepath.Join(moduleDir, ".pdkignore")},
		{FileName: "module/rubocop.yml", Destination: filepath.Join(moduleDir, ".rubocop.yml")},
		{FileName: "module/hiera.yaml", Destination: filepath.Join(moduleDir, "hiera.yaml")},
		{FileName: "module/devcontainer/devcontainer.json", Destination: filepath.Join(moduleDir, ".devcontainer", "devcontainer.json")},
		{FileName: "module/spec/spec_helper.rb", Destination: filepath.Join(moduleDir, "spec", "spec_helper.rb")},
		{FileName: "module/spec/spec_helper_acceptance.rb", Destination: filepath.Join(moduleDir, "spec", "spec_helper_acceptance.rb")},
		{FileName: "module/spec/acceptance/init_spec.rb", Destination: filepath.Join(moduleDir, "spec", "acceptance", "init_spec.rb")},
		{FileName: "module/spec/default_facts.yml", Destination: filepath.Join(moduleDir, "spec", "default_facts.yml")},
		{FileName: "module/data/common.yaml", Destination: filepath.Join(moduleDir, "data", "common.yaml")},
		{FileName: "common/gitkeep", Destination: filepath.Join(moduleDir, "data", ".gitkeep")},
		{FileName: "common/gitkeep", Destination: filepath.Join(moduleDir, "examples", ".gitkeep")},
		{FileName: "common/gitkeep", Destination: filepath.Join(moduleDir, "files", ".gitkeep")},
		{FileName: "common/gitkeep", Destination: filepath.Join(moduleDir, "tasks", ".gitkeep")},
		{FileName: "common/gitkeep", Destination: filepath.Join(moduleDir, "templates", ".gitkeep")},
	}

	data := struct {
		ModuleName string
		ForgeUser  string
		Author     string
		License    string
		ClassName  string
	}{
		ModuleName: opts.Name,
		ForgeUser:  opts.ForgeUser,
		Author:     opts.Author,
		License:    opts.License,
		ClassName:  opts.Name,
	}

	if err := RenderTemplates(renderer, templates, data, opts.Force); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	fmt.Printf("Created new module %s in %s\n", opts.Name, moduleDir)
	return nil
}
