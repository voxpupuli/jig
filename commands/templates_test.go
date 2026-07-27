// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/voxpupuli/jig/internal/config"
)

// runTemplatesResolve executes `jig templates resolve <name>` with the given
// extra flags and returns the command output.
func runTemplatesResolve(t *testing.T, a *App, name string, flags ...string) (string, error) {
	t.Helper()
	cmd := a.templatesCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(append(append([]string{"resolve"}, flags...), name))
	err := cmd.Execute()
	return buf.String(), err
}

// With no external source configured, resolution must report the embedded
// templates and where the name resolved.
func TestTemplatesResolve_Embedded(t *testing.T) {
	t.Chdir(t.TempDir())
	a := testApp(config.Config{})

	out, err := runTemplatesResolve(t, a, "class/class.pp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "embedded templates only") {
		t.Errorf("expected embedded-only notice, got: %q", out)
	}
	if !strings.Contains(out, "resolved class/class.pp to embedded template templates/class/class.pp.tmpl") {
		t.Errorf("expected embedded resolution line, got: %q", out)
	}
}

// An external directory given via --template-dir must be reported with its
// provenance, show the paths checked, and win over the embedded template.
func TestTemplatesResolve_ExternalDir(t *testing.T) {
	t.Chdir(t.TempDir())
	dir := t.TempDir()
	path := filepath.Join(dir, "class", "class.pp.tmpl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("custom"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := testApp(config.Config{})
	out, err := runTemplatesResolve(t, a, "class/class.pp", "--template-dir", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "external template directory: "+dir) {
		t.Errorf("expected external dir line, got: %q", out)
	}
	if !strings.Contains(out, "--template-dir flag") {
		t.Errorf("expected provenance in output, got: %q", out)
	}
	if !strings.Contains(out, "resolved class/class.pp to external template "+path) {
		t.Errorf("expected external resolution line, got: %q", out)
	}
}

// A miss in the external directory must show the fallback to the embedded
// templates, mirroring what rendering would do.
func TestTemplatesResolve_ExternalFallback(t *testing.T) {
	t.Chdir(t.TempDir())
	dir := t.TempDir()

	a := testApp(config.Config{TemplateDir: dir})
	out, err := runTemplatesResolve(t, a, "class/class.pp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "template_dir from config") {
		t.Errorf("expected config provenance, got: %q", out)
	}
	if !strings.Contains(out, "falling back to embedded templates") {
		t.Errorf("expected fallback notice, got: %q", out)
	}
	if !strings.Contains(out, "resolved class/class.pp to embedded template") {
		t.Errorf("expected embedded resolution line, got: %q", out)
	}
}

// A name that exists nowhere must fail with a non-zero exit after printing
// every path that was checked.
func TestTemplatesResolve_NotFound(t *testing.T) {
	t.Chdir(t.TempDir())
	a := testApp(config.Config{})

	out, err := runTemplatesResolve(t, a, "nonexistent/nothing.pp")
	if err == nil {
		t.Fatal("expected an error for an unresolvable template name")
	}
	if !strings.Contains(out, "looking for templates/nonexistent/nothing.pp.tmpl (embedded) ... not found") {
		t.Errorf("expected the checked paths in output, got: %q", out)
	}
}

// runTemplatesDump executes `jig templates dump <destination>` and returns
// the command output.
func runTemplatesDump(t *testing.T, a *App, destination string) (string, error) {
	t.Helper()
	cmd := a.templatesCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"dump", destination})
	err := cmd.Execute()
	return buf.String(), err
}

// Dumping to "." must write into the current directory in place rather than
// trying to rename it out of the way: renaming "." fails on every platform,
// and dumping is expected to behave like unpacking an archive alongside
// existing files. See https://github.com/voxpupuli/jig/issues/82.
func TestTemplatesDump_CurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "another-file"), []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := testApp(config.Config{})
	if _, err := runTemplatesDump(t, a, "."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "another-file")); err != nil {
		t.Errorf("expected pre-existing file to survive the dump: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "class", "class.pp.tmpl")); err != nil {
		t.Errorf("expected templates to be written into the current directory: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(dir))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".bak.") {
			t.Errorf("expected no backup directory to be created, found %q", e.Name())
		}
	}
}

// Dumping to a non-cwd destination that already exists must still back it up
// with a timestamp suffix before writing.
func TestTemplatesDump_BacksUpExistingDestination(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir("existing", 0o755); err != nil {
		t.Fatal(err)
	}

	a := testApp(config.Config{})
	if _, err := runTemplatesDump(t, a, "existing"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var sawBackup bool
	for _, e := range entries {
		if strings.Contains(e.Name(), "existing.bak.") {
			sawBackup = true
		}
	}
	if !sawBackup {
		t.Error("expected a backup directory to be created")
	}
}

// Inside a module directory, the [template] section of jig.toml must feed
// resolution, matching what component commands do.
func TestTemplatesResolve_ModuleConfigURL(t *testing.T) {
	repo := templateRepo(t)
	moduleDir := t.TempDir()
	moduleConfig := config.ModuleConfig{Template: config.ModuleTemplate{URL: repo}}
	if err := moduleConfig.Write(moduleDir); err != nil {
		t.Fatalf("write jig.toml: %v", err)
	}
	t.Chdir(moduleDir)

	a := testApp(config.Config{})
	out, err := runTemplatesResolve(t, a, "class/class.pp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "external templates: "+repo) {
		t.Errorf("expected remote source line, got: %q", out)
	}
	if !strings.Contains(out, "[template] section of") {
		t.Errorf("expected jig.toml provenance, got: %q", out)
	}
	if !strings.Contains(out, "resolved class/class.pp to embedded template") {
		t.Errorf("expected embedded fallback resolution, got: %q", out)
	}
}
