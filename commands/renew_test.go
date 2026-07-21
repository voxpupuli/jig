// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/voxpupuli/jig/internal/config"
	"github.com/voxpupuli/jig/internal/module"
)

// moduleTemplateRepo builds a local git repository holding a module template
// tree, usable as a template url without any network.
func moduleTemplateRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init template repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	for name, content := range files {
		path := filepath.Join(dir, "module", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := wt.Add(filepath.ToSlash(filepath.Join("module", name))); err != nil {
			t.Fatalf("add %s: %v", name, err)
		}
	}
	_, err = wt.Commit("templates", &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	return dir
}

// renewModuleDir creates a module directory with metadata.json and a jig.toml
// carrying the given template url and renew allowlist, and chdirs into it.
func renewModuleDir(t *testing.T, templateURL string, paths []string) string {
	t.Helper()
	dir := t.TempDir()
	meta := module.NewMetadata("mymodule", "myuser", "My Name")
	if err := meta.Write(filepath.Join(dir, "metadata.json")); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	cfg := config.ModuleConfig{
		Template: config.ModuleTemplate{URL: templateURL},
		Renew:    config.RenewConfig{Paths: paths},
	}
	if err := cfg.Write(dir); err != nil {
		t.Fatalf("write jig.toml: %v", err)
	}
	t.Chdir(dir)
	return dir
}

// runRenew executes `jig renew` with the given extra arguments and returns
// the combined command output.
func runRenew(t *testing.T, a *App, args ...string) (string, error) {
	t.Helper()
	cmd := a.renewCmd()
	cmd.SetArgs(args)
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	err := cmd.Execute()
	return out.String(), err
}

// With an empty [renew] allowlist the command must refuse to do anything and
// say why, so template changes can never overwrite work by accident.
func TestRenewCmd_EmptyAllowlist(t *testing.T) {
	renewModuleDir(t, "", nil)

	_, err := runRenew(t, testApp(config.Config{}))
	if err == nil || !strings.Contains(err.Error(), "[renew]") {
		t.Fatalf("expected an error pointing at the [renew] allowlist, got: %v", err)
	}
}

// A full renew from the recorded template url must overwrite the allowlisted
// file and record the fetched commit in jig.toml.
func TestRenewCmd_RenewsAndRecordsCommit(t *testing.T) {
	repo := moduleTemplateRepo(t, map[string]string{
		"README.md.tmpl": "# {{.ModuleName}} v2\n",
	})
	dir := renewModuleDir(t, repo, []string{"README.md"})
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runRenew(t, testApp(config.Config{}))
	if err != nil {
		t.Fatalf("unexpected error: %v (output: %q)", err, out)
	}

	content, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "# mymodule v2\n" {
		t.Errorf("README.md: got %q, want rendered template output", content)
	}

	cfg, err := config.LoadModuleConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Template.URL != repo {
		t.Errorf("recorded template url: got %q, want %q", cfg.Template.URL, repo)
	}
	if cfg.Template.Commit == "" {
		t.Error("renew from a remote must record the fetched commit in jig.toml")
	}
	if cfg.Renew.Paths == nil {
		t.Error("rewriting jig.toml must preserve the renew allowlist")
	}
}

// --dry-run must neither touch module files nor update the recorded commit.
func TestRenewCmd_DryRunChangesNothing(t *testing.T) {
	repo := moduleTemplateRepo(t, map[string]string{
		"README.md.tmpl": "# {{.ModuleName}} v2\n",
	})
	dir := renewModuleDir(t, repo, []string{"README.md"})
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runRenew(t, testApp(config.Config{}), "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v (output: %q)", err, out)
	}
	if !strings.Contains(out, "would renew README.md") {
		t.Errorf("dry-run output should report the file, got: %q", out)
	}

	content, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "# old\n" {
		t.Errorf("dry run must not modify files, README.md became %q", content)
	}

	cfg, err := config.LoadModuleConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Template.Commit != "" {
		t.Errorf("dry run must not record a commit, got %q", cfg.Template.Commit)
	}
}

// The renew command must accept the shared template source flags on itself,
// since it has no parent command carrying them.
func TestRenewCmd_TemplateDirFlag(t *testing.T) {
	tmplDir := t.TempDir()
	path := filepath.Join(tmplDir, "module", "README.md.tmpl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# {{.ModuleName}} from dir\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := renewModuleDir(t, "", []string{"README.md"})

	out, err := runRenew(t, testApp(config.Config{}), "--template-dir", tmplDir)
	if err != nil {
		t.Fatalf("unexpected error: %v (output: %q)", err, out)
	}

	content, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "# mymodule from dir\n" {
		t.Errorf("README.md: got %q, want output rendered from --template-dir", content)
	}
}
