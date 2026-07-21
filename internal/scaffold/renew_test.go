// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeModuleFile writes a file into a module directory, creating parent
// directories as needed.
func writeModuleFile(t *testing.T, moduleDir, name, content string) {
	t.Helper()
	path := filepath.Join(moduleDir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readModuleFile(t *testing.T, moduleDir, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(moduleDir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// An allowlisted file whose rendered content differs must be overwritten.
func TestRenew_OverwritesAllowlistedFile(t *testing.T) {
	tmplDir := t.TempDir()
	writeModuleTemplate(t, tmplDir, "README.md.tmpl", "# {{.ModuleName}} v2\n")

	moduleDir := makeModuleDir(t, "myuser", "mymodule")
	writeModuleFile(t, moduleDir, "README.md", "# old\n")

	var out strings.Builder
	err := Renew(RenewOptions{
		ModuleDir:   moduleDir,
		TemplateDir: tmplDir,
		Paths:       []string{"README.md"},
		Out:         &out,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := readModuleFile(t, moduleDir, "README.md"); got != "# mymodule v2\n" {
		t.Errorf("README.md: got %q, want rendered template output", got)
	}
	if !strings.Contains(out.String(), "renewed README.md") {
		t.Errorf("output should report the renewed file, got: %q", out.String())
	}
}

// Files not matching the allowlist must never be touched, even when the
// template tree has newer content for them.
func TestRenew_SkipsFilesNotAllowlisted(t *testing.T) {
	tmplDir := t.TempDir()
	writeModuleTemplate(t, tmplDir, "README.md.tmpl", "# new\n")
	writeModuleTemplate(t, tmplDir, "Gemfile", "gems v2\n")

	moduleDir := makeModuleDir(t, "myuser", "mymodule")
	writeModuleFile(t, moduleDir, "Gemfile", "old gems\n")

	var out strings.Builder
	err := Renew(RenewOptions{
		ModuleDir:   moduleDir,
		TemplateDir: tmplDir,
		Paths:       []string{"README.md"},
		Out:         &out,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := readModuleFile(t, moduleDir, "Gemfile"); got != "old gems\n" {
		t.Errorf("Gemfile is not allowlisted and must not change, got %q", got)
	}
}

// DryRun must leave every file untouched and print a diff of what would
// change.
func TestRenew_DryRunShowsDiffWithoutWriting(t *testing.T) {
	tmplDir := t.TempDir()
	writeModuleTemplate(t, tmplDir, "README.md.tmpl", "# {{.ModuleName}} v2\n")

	moduleDir := makeModuleDir(t, "myuser", "mymodule")
	writeModuleFile(t, moduleDir, "README.md", "# old\n")

	var out strings.Builder
	err := Renew(RenewOptions{
		ModuleDir:   moduleDir,
		TemplateDir: tmplDir,
		Paths:       []string{"README.md"},
		DryRun:      true,
		Out:         &out,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := readModuleFile(t, moduleDir, "README.md"); got != "# old\n" {
		t.Errorf("dry run must not modify files, README.md became %q", got)
	}
	got := out.String()
	for _, want := range []string{"would renew README.md", "-# old", "+# mymodule v2", "1 file(s) would be renewed"} {
		if !strings.Contains(got, want) {
			t.Errorf("dry-run output should contain %q, got: %q", want, got)
		}
	}
}

// A matched file whose content already equals the rendered output is
// reported as up to date, not rewritten.
func TestRenew_UpToDateFileIsNotRewritten(t *testing.T) {
	tmplDir := t.TempDir()
	writeModuleTemplate(t, tmplDir, "README.md.tmpl", "# {{.ModuleName}}\n")

	moduleDir := makeModuleDir(t, "myuser", "mymodule")
	writeModuleFile(t, moduleDir, "README.md", "# mymodule\n")

	var out strings.Builder
	err := Renew(RenewOptions{
		ModuleDir:   moduleDir,
		TemplateDir: tmplDir,
		Paths:       []string{"README.md"},
		Out:         &out,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "0 file(s) renewed, 1 already up to date") {
		t.Errorf("expected an up-to-date summary, got: %q", out.String())
	}
}

// An allowlisted file the module does not have yet (added to the template
// tree after scaffolding) must be created, including parent directories.
func TestRenew_CreatesMissingAllowlistedFile(t *testing.T) {
	tmplDir := t.TempDir()
	writeModuleTemplate(t, tmplDir, "docs/guide.md.tmpl", "guide for {{.ModuleName}}\n")

	moduleDir := makeModuleDir(t, "myuser", "mymodule")

	var out strings.Builder
	err := Renew(RenewOptions{
		ModuleDir:   moduleDir,
		TemplateDir: tmplDir,
		Paths:       []string{"docs/**"},
		Out:         &out,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := readModuleFile(t, moduleDir, "docs/guide.md"); got != "guide for mymodule\n" {
		t.Errorf("docs/guide.md: got %q, want rendered template output", got)
	}
}

// metadata.json and jig.toml are generated by jig and stay reserved: even an
// allowlist naming them must not overwrite them.
func TestRenew_ReservedFilesAreNeverRenewed(t *testing.T) {
	tmplDir := t.TempDir()
	writeModuleTemplate(t, tmplDir, "metadata.json", "{}\n")

	moduleDir := makeModuleDir(t, "myuser", "mymodule")
	before := readModuleFile(t, moduleDir, "metadata.json")

	var out strings.Builder
	err := Renew(RenewOptions{
		ModuleDir:   moduleDir,
		TemplateDir: tmplDir,
		Paths:       []string{"metadata.json"},
		Out:         &out,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := readModuleFile(t, moduleDir, "metadata.json"); got != before {
		t.Errorf("metadata.json must never be renewed, got %q", got)
	}
	if !strings.Contains(out.String(), "no template files match") {
		t.Errorf("expected the no-match summary, got: %q", out.String())
	}
}

// A module directory without metadata.json is not a module; renew must fail
// rather than render against empty metadata.
func TestRenew_RequiresModuleDirectory(t *testing.T) {
	err := Renew(RenewOptions{
		ModuleDir: t.TempDir(),
		Paths:     []string{"README.md"},
	})
	if err == nil {
		t.Fatal("expected an error outside a module directory")
	}
}
