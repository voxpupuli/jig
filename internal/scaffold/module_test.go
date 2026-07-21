// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/voxpupuli/jig/internal/config"
)

func TestNewModule(t *testing.T) {
	t.Run("creates module with expected files", func(t *testing.T) {
		dir := t.TempDir()
		opts := Options{
			ForgeUser: "myuser",
			Name:      "mymodule",
			Author:    "My Name",
			License:   "Apache-2.0",
			Summary:   "A test module",
			Source:    "https://example.com",
			TargetDir: dir,
		}

		if err := NewModule(opts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		moduleDir := filepath.Join(dir, "mymodule")
		expectedFiles := []string{
			"metadata.json",
			"jig.toml",
			"manifests/init.pp",
			"README.md",
			"CHANGELOG.md",
			"Gemfile",
			"Rakefile",
			".editorconfig",
			".gitignore",
			".overcommit.yml",
			".rubocop.yml",
			".devcontainer/devcontainer.json",
			"hiera.yaml",
			"spec/classes/init_spec.rb",
			"spec/spec_helper.rb",
			"spec/spec_helper_acceptance.rb",
			"spec/acceptance/init_spec.rb",
			"spec/default_facts.yml",
			"data/common.yaml",
			"data/.gitkeep",
			"examples/.gitkeep",
			"files/.gitkeep",
			"tasks/.gitkeep",
			"templates/.gitkeep",
		}
		for _, f := range expectedFiles {
			path := filepath.Join(moduleDir, f)
			if _, err := os.Stat(path); err != nil {
				t.Errorf("expected file %s to exist: %v", f, err)
			}
		}
	})

	t.Run("unmapped files in an external template dir are written", func(t *testing.T) {
		dir := t.TempDir()
		tmplDir := t.TempDir()
		// Neither of these paths has (or needs) a mapping in jig code (#36):
		// a verbatim file with GitHub Actions syntax and a rendered .tmpl.
		writeModuleTemplate(t, tmplDir, ".github/workflows/ci.yml", "run: ${{ matrix.ruby }}\n")
		writeModuleTemplate(t, tmplDir, "CONTRIBUTING.md.tmpl", "# Contributing to {{.ModuleName}}\n")

		opts := Options{
			ForgeUser:   "myuser",
			Name:        "mymodule",
			Author:      "My Name",
			License:     "Apache-2.0",
			TargetDir:   dir,
			TemplateDir: tmplDir,
		}
		if err := NewModule(opts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		moduleDir := filepath.Join(dir, "mymodule")
		ci, err := os.ReadFile(filepath.Join(moduleDir, ".github", "workflows", "ci.yml"))
		if err != nil {
			t.Fatalf("expected unmapped verbatim file to be written: %v", err)
		}
		if string(ci) != "run: ${{ matrix.ruby }}\n" {
			t.Errorf("verbatim file must not be rendered, got %q", ci)
		}

		contributing, err := os.ReadFile(filepath.Join(moduleDir, "CONTRIBUTING.md"))
		if err != nil {
			t.Fatalf("expected unmapped .tmpl file to be written: %v", err)
		}
		if string(contributing) != "# Contributing to mymodule\n" {
			t.Errorf(".tmpl file must be rendered with .tmpl stripped, got %q", contributing)
		}

		// Embedded files still arrive alongside the external additions.
		if _, err := os.Stat(filepath.Join(moduleDir, ".gitignore")); err != nil {
			t.Errorf("expected embedded .gitignore alongside external additions: %v", err)
		}
	})

	t.Run("external template overrides embedded file at same destination", func(t *testing.T) {
		dir := t.TempDir()
		tmplDir := t.TempDir()
		// Plain file overrides the embedded README.md.tmpl by destination.
		writeModuleTemplate(t, tmplDir, "README.md", "custom readme\n")

		opts := Options{
			ForgeUser:   "myuser",
			Name:        "mymodule",
			Author:      "My Name",
			License:     "Apache-2.0",
			TargetDir:   dir,
			TemplateDir: tmplDir,
		}
		if err := NewModule(opts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		readme, err := os.ReadFile(filepath.Join(dir, "mymodule", "README.md"))
		if err != nil {
			t.Fatal(err)
		}
		if string(readme) != "custom readme\n" {
			t.Errorf("expected external README.md to win, got %q", readme)
		}
	})

	t.Run("reserved files in the template tree are ignored", func(t *testing.T) {
		dir := t.TempDir()
		tmplDir := t.TempDir()
		writeModuleTemplate(t, tmplDir, "metadata.json", `{"name":"evil"}`)
		writeModuleTemplate(t, tmplDir, "jig.toml", "[template]\nurl = \"evil\"\n")

		opts := Options{
			ForgeUser:   "myuser",
			Name:        "mymodule",
			Author:      "My Name",
			License:     "Apache-2.0",
			TargetDir:   dir,
			TemplateDir: tmplDir,
		}
		if err := NewModule(opts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m, err := GetMetadata(filepath.Join(dir, "mymodule"))
		if err != nil {
			t.Fatalf("could not read generated metadata: %v", err)
		}
		if m.Name != "myuser-mymodule" {
			t.Errorf("metadata.json must come from jig, not the template tree; got name %q", m.Name)
		}

		cfg, err := config.LoadModuleConfig(filepath.Join(dir, "mymodule"))
		if err != nil {
			t.Fatalf("could not read generated jig.toml: %v", err)
		}
		if cfg.Template.URL == "evil" {
			t.Error("jig.toml must come from jig, not the template tree")
		}
	})

	t.Run("conflicting template variants fail the scaffold", func(t *testing.T) {
		dir := t.TempDir()
		tmplDir := t.TempDir()
		writeModuleTemplate(t, tmplDir, "foo.yml", "plain\n")
		writeModuleTemplate(t, tmplDir, "foo.yml.tmpl", "templated\n")

		opts := Options{
			ForgeUser:   "myuser",
			Name:        "mymodule",
			Author:      "My Name",
			License:     "Apache-2.0",
			TargetDir:   dir,
			TemplateDir: tmplDir,
		}
		if err := NewModule(opts); err == nil {
			t.Error("expected error for foo.yml alongside foo.yml.tmpl, got nil")
		}
	})

	t.Run("returns error when directory exists without force", func(t *testing.T) {
		dir := t.TempDir()
		moduleDir := filepath.Join(dir, "mymodule")
		if err := os.Mkdir(moduleDir, 0755); err != nil {
			t.Fatal(err)
		}

		opts := Options{
			Name:      "mymodule",
			TargetDir: dir,
		}

		if err := NewModule(opts); err == nil {
			t.Error("expected error for existing directory without --force, got nil")
		}
	})

	t.Run("backs up existing directory when force is set", func(t *testing.T) {
		dir := t.TempDir()
		moduleDir := filepath.Join(dir, "mymodule")
		if err := os.Mkdir(moduleDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(moduleDir, "sentinel"), []byte("old"), 0644); err != nil {
			t.Fatal(err)
		}

		opts := Options{
			ForgeUser: "myuser",
			Name:      "mymodule",
			Author:    "My Name",
			License:   "Apache-2.0",
			Summary:   "A test module",
			Source:    "https://example.com",
			Force:     true,
			TargetDir: dir,
		}

		if err := NewModule(opts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(filepath.Join(moduleDir, "metadata.json")); err != nil {
			t.Error("expected metadata.json in new module dir")
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		var backupFound bool
		for _, e := range entries {
			if e.Name() != "mymodule" && len(e.Name()) > len("mymodule") {
				backupFound = true
				break
			}
		}
		if !backupFound {
			t.Error("expected a backup directory to exist after --force")
		}
	})

	t.Run("metadata.json contains correct name", func(t *testing.T) {
		dir := t.TempDir()
		opts := Options{
			ForgeUser: "myuser",
			Name:      "mymodule",
			Author:    "My Name",
			License:   "Apache-2.0",
			Summary:   "A test module",
			Source:    "https://example.com",
			TargetDir: dir,
		}

		if err := NewModule(opts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m, err := GetMetadata(filepath.Join(dir, "mymodule"))
		if err != nil {
			t.Fatalf("could not read generated metadata: %v", err)
		}
		if m.Name != "myuser-mymodule" {
			t.Errorf("Name: got %q, want %q", m.Name, "myuser-mymodule")
		}
	})

	t.Run("template provenance is recorded in jig.toml, not metadata.json", func(t *testing.T) {
		dir := t.TempDir()
		opts := Options{
			ForgeUser:      "myuser",
			Name:           "mymodule",
			Author:         "My Name",
			License:        "Apache-2.0",
			Summary:        "A test module",
			Source:         "https://example.com",
			TargetDir:      dir,
			TemplateURL:    "ssh://git@example.com/templates.git",
			TemplateRef:    "main",
			TemplateCommit: "abc123",
		}

		if err := NewModule(opts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		moduleDir := filepath.Join(dir, "mymodule")
		cfg, err := config.LoadModuleConfig(moduleDir)
		if err != nil {
			t.Fatalf("could not read generated jig.toml: %v", err)
		}
		if cfg.Template.URL != opts.TemplateURL || cfg.Template.Ref != opts.TemplateRef || cfg.Template.Commit != opts.TemplateCommit {
			t.Errorf("jig.toml template section: got %+v", cfg.Template)
		}

		m, err := GetMetadata(moduleDir)
		if err != nil {
			t.Fatalf("could not read generated metadata: %v", err)
		}
		if m.TemplateURL != "" || m.TemplateRef != "" || m.TemplateCommit != "" {
			t.Errorf("metadata.json must not record template provenance anymore, got url=%q ref=%q commit=%q",
				m.TemplateURL, m.TemplateRef, m.TemplateCommit)
		}
	})
}

func TestNewModule_PathTraversal(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name       string
		moduleName string
	}{
		{"path separator", "foo/bar"},
		{"dot dot", ".."},
		{"dot dot slash", "../evil"},
		{"empty", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := Options{
				ForgeUser: "myuser",
				Name:      tc.moduleName,
				Author:    "My Name",
				License:   "Apache-2.0",
				Summary:   "test",
				Source:    "https://example.com",
				TargetDir: dir,
			}
			err := NewModule(opts)
			if err == nil {
				t.Errorf("expected error for module name %q, got nil", tc.moduleName)
			}
		})
	}
}
