// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/voxpupuli/jig/v2/internal/config"
	"github.com/voxpupuli/jig/v2/internal/module"
)

// runConvert executes `jig convert` with the given extra arguments and
// returns the combined command output.
func runConvert(t *testing.T, a *App, args ...string) (string, error) {
	t.Helper()
	cmd := a.convertCmd()
	cmd.SetArgs(args)
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	err := cmd.Execute()
	return out.String(), err
}

// With no metadata.json, --skip-interview plus flags and config must be
// enough to create it without prompting -- there is no stdin in tests.
func TestConvertCmd_MissingMetadata_SkipInterviewUsesFlagsAndConfig(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(filepath.Join(dir))
	moduleDir := filepath.Join(dir, "acme-widget")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(moduleDir)

	out, err := runConvert(t, testApp(config.Config{ForgeUsername: "acme", Author: "Ada"}), "--skip-interview", "-s", "widgets", "-S", "https://example.com/widget")
	if err != nil {
		t.Fatalf("unexpected error: %v (output: %q)", err, out)
	}

	meta, err := module.ReadMetadata(filepath.Join(moduleDir, "metadata.json"))
	if err != nil {
		t.Fatalf("failed to read metadata.json: %v", err)
	}
	if meta.Name != "acme-widget" {
		t.Errorf("name: got %q, want %q", meta.Name, "acme-widget")
	}
	if meta.Author != "Ada" {
		t.Errorf("author: got %q, want %q", meta.Author, "Ada")
	}
	if meta.Summary != "widgets" {
		t.Errorf("summary: got %q, want %q", meta.Summary, "widgets")
	}

	if _, err := os.Stat(filepath.Join(moduleDir, config.ModuleConfigFileName)); err != nil {
		t.Errorf("expected jig.toml to be created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(moduleDir, "Gemfile")); err != nil {
		t.Errorf("expected Gemfile to be created: %v", err)
	}
}

// An explicit flag must win over both the Modulefile and the config default.
func TestConvertCmd_MissingMetadata_FlagOverridesModulefile(t *testing.T) {
	dir := t.TempDir()
	moduleDir := filepath.Join(dir, "whatever")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	modulefile := "name 'binford2k-demo'\nauthor 'Modulefile Author'\n"
	if err := os.WriteFile(filepath.Join(moduleDir, "Modulefile"), []byte(modulefile), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(moduleDir)

	out, err := runConvert(t, testApp(config.Config{}), "--skip-interview", "-a", "Flag Author")
	if err != nil {
		t.Fatalf("unexpected error: %v (output: %q)", err, out)
	}

	meta, err := module.ReadMetadata(filepath.Join(moduleDir, "metadata.json"))
	if err != nil {
		t.Fatalf("failed to read metadata.json: %v", err)
	}
	if meta.Name != "binford2k-demo" {
		t.Errorf("name should come from the Modulefile: got %q", meta.Name)
	}
	if meta.Author != "Flag Author" {
		t.Errorf("author flag should override the Modulefile: got %q", meta.Author)
	}

	if _, err := os.Stat(filepath.Join(moduleDir, "Modulefile")); err != nil {
		t.Errorf("Modulefile should be left in place: %v", err)
	}
}

// Invalid JSON in an existing metadata.json must stop convert before it
// touches any other file.
func TestConvertCmd_InvalidMetadataJSON_Stops(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	_, err := runConvert(t, testApp(config.Config{}))
	if err == nil {
		t.Fatal("expected an error for invalid metadata.json")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "Gemfile")); !os.IsNotExist(statErr) {
		t.Error("Gemfile should not be written when metadata.json fails to parse")
	}
}

// --dry-run on a module missing metadata.json must not write anything.
func TestConvertCmd_DryRun_MissingMetadata(t *testing.T) {
	dir := t.TempDir()
	moduleDir := filepath.Join(dir, "acme-widget")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(moduleDir)

	out, err := runConvert(t, testApp(config.Config{ForgeUsername: "acme", Author: "Ada"}), "--skip-interview", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v (output: %q)", err, out)
	}
	if !strings.Contains(out, "would create") {
		t.Errorf("expected dry-run output to describe what would be created, got: %q", out)
	}
	if _, statErr := os.Stat(filepath.Join(moduleDir, "metadata.json")); !os.IsNotExist(statErr) {
		t.Error("dry-run must not create metadata.json")
	}
}
