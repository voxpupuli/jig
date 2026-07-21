// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/voxpupuli/jig/internal/config"
	"github.com/voxpupuli/jig/internal/module"
)

func testApp(cfg config.Config) *App {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return &App{Config: cfg, Logger: logger}
}

// newSubCmd returns the named `jig new` subcommand with its parent linked, so
// InheritedFlags resolves the persistent template flags. The parent is
// returned too, for setting those flags.
func newSubCmd(t *testing.T, a *App, name string) (*cobra.Command, *cobra.Command) {
	t.Helper()
	parent := a.newCmd()
	cmd, _, err := parent.Find([]string{name})
	if err != nil {
		t.Fatalf("find subcommand %s: %v", name, err)
	}
	return cmd, parent
}

// templateRepo builds a minimal local git repository usable as a
// --template-url target without any network.
func templateRepo(t *testing.T) string {
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
	if err := os.WriteFile(filepath.Join(dir, "marker.txt"), []byte("templates"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	if _, err := wt.Add("marker.txt"); err != nil {
		t.Fatalf("add marker: %v", err)
	}
	_, err = wt.Commit("templates", &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	return dir
}

// With no flags, no metadata, and no config, the template source must be the
// embedded templates (empty Dir).
func TestResolveTemplateSource_Embedded(t *testing.T) {
	a := testApp(config.Config{})
	cmd, _ := newSubCmd(t, a, "class")

	src, err := a.resolveTemplateSource(cmd.InheritedFlags(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer src.Cleanup()
	if src.Dir != "" || src.URL != "" {
		t.Errorf("expected embedded templates, got %+v", src)
	}
}

// template_dir from config must be used when no flags are set, and the
// --template-dir flag must override it.
func TestResolveTemplateSource_DirPrecedence(t *testing.T) {
	a := testApp(config.Config{TemplateDir: "/from-config"})
	cmd, parent := newSubCmd(t, a, "class")

	src, err := a.resolveTemplateSource(cmd.InheritedFlags(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src.Dir != "/from-config" {
		t.Errorf("Dir: got %q, want config value", src.Dir)
	}

	if err := parent.PersistentFlags().Set("template-dir", "/from-flag"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	src, err = a.resolveTemplateSource(cmd.InheritedFlags(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src.Dir != "/from-flag" {
		t.Errorf("Dir: got %q, want flag value", src.Dir)
	}
}

// --template-url and --template-dir together must be rejected.
func TestResolveTemplateSource_URLAndDirConflict(t *testing.T) {
	a := testApp(config.Config{})
	cmd, parent := newSubCmd(t, a, "class")

	parent.PersistentFlags().Set("template-url", "/some/repo")
	parent.PersistentFlags().Set("template-dir", "/some/dir")

	if _, err := a.resolveTemplateSource(cmd.InheritedFlags(), ""); err == nil {
		t.Fatal("expected an error for --template-url with --template-dir")
	}
}

// --template-ref without any template URL (flag or metadata) must be rejected
// rather than silently ignored.
func TestResolveTemplateSource_RefWithoutURL(t *testing.T) {
	a := testApp(config.Config{})
	cmd, parent := newSubCmd(t, a, "class")

	parent.PersistentFlags().Set("template-ref", "main")

	if _, err := a.resolveTemplateSource(cmd.InheritedFlags(), ""); err == nil {
		t.Fatal("expected an error for --template-ref without --template-url")
	}
}

// A --template-url flag must produce a temporary clone and report the
// provenance that `new module` records in metadata.json.
func TestResolveTemplateSource_URLFlag(t *testing.T) {
	repo := templateRepo(t)
	a := testApp(config.Config{})
	cmd, parent := newSubCmd(t, a, "module")

	parent.PersistentFlags().Set("template-url", repo)

	src, err := a.resolveTemplateSource(cmd.InheritedFlags(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer src.Cleanup()

	if _, err := os.Stat(filepath.Join(src.Dir, "marker.txt")); err != nil {
		t.Errorf("clone should contain marker.txt: %v", err)
	}
	if src.URL != repo {
		t.Errorf("URL: got %q, want %q", src.URL, repo)
	}
	if src.Commit == "" {
		t.Error("Commit should be recorded for a remote fetch")
	}

	dir := src.Dir
	src.Cleanup()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("Cleanup should remove the temporary clone %s", dir)
	}
}

// Inside a module whose jig.toml records a template url, component commands
// must fetch from that URL with no flags at all -- the shared-team case from
// issue #60. A template-url left in metadata.json by jig 1.x must not
// interfere.
func TestResolveTemplateSource_ModuleConfigURL(t *testing.T) {
	repo := templateRepo(t)

	moduleDir := t.TempDir()
	meta := module.NewMetadata("mymodule", "myuser", "My Name")
	meta.TemplateURL = "/nonexistent-1x-recorded-url"
	if err := meta.Write(filepath.Join(moduleDir, "metadata.json")); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	moduleConfig := config.ModuleConfig{Template: config.ModuleTemplate{URL: repo}}
	if err := moduleConfig.Write(moduleDir); err != nil {
		t.Fatalf("write jig.toml: %v", err)
	}

	a := testApp(config.Config{})
	cmd, _ := newSubCmd(t, a, "class")

	src, err := a.resolveTemplateSource(cmd.InheritedFlags(), moduleDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer src.Cleanup()
	if src.URL != repo {
		t.Errorf("URL: got %q, want the jig.toml url %q", src.URL, repo)
	}
	if _, err := os.Stat(filepath.Join(src.Dir, "marker.txt")); err != nil {
		t.Errorf("clone from jig.toml template url should contain marker.txt: %v", err)
	}
}

// An unparseable jig.toml must fail template resolution loudly rather than
// silently falling back to other sources.
func TestResolveTemplateSource_InvalidModuleConfig(t *testing.T) {
	moduleDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleDir, config.ModuleConfigFileName), []byte("not = toml = ["), 0o644); err != nil {
		t.Fatal(err)
	}

	a := testApp(config.Config{})
	cmd, _ := newSubCmd(t, a, "class")

	if _, err := a.resolveTemplateSource(cmd.InheritedFlags(), moduleDir); err == nil {
		t.Fatal("expected an error for invalid jig.toml")
	}
}

// Template settings in metadata.json (written by jig 1.x) are not supported:
// the recorded URL must be ignored -- resolution falls through to the
// embedded templates -- and a warning must tell the user to move the
// settings to jig.toml.
func TestResolveTemplateSource_MetadataNotSupported(t *testing.T) {
	moduleDir := t.TempDir()
	meta := module.NewMetadata("mymodule", "myuser", "My Name")
	meta.TemplateURL = "/nonexistent-1x-recorded-url"
	if err := meta.Write(filepath.Join(moduleDir, "metadata.json")); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	a := testApp(config.Config{})
	cmd, _ := newSubCmd(t, a, "class")

	out := captureStdout(t, func() {
		src, err := a.resolveTemplateSource(cmd.InheritedFlags(), moduleDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer src.Cleanup()
		if src.Dir != "" || src.URL != "" {
			t.Errorf("metadata template-url must be ignored, expected embedded templates, got %+v", src)
		}
	})

	if !strings.Contains(out, "warning") || !strings.Contains(out, "metadata.json") || !strings.Contains(out, "jig.toml") {
		t.Errorf("expected a warning pointing from metadata.json to jig.toml, got output: %q", out)
	}
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns
// everything written, since resolveTemplateSource reports warnings via
// fmt.Println.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				sb.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		done <- sb.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = orig
	return <-done
}
