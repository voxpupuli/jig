// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadModuleConfig_MissingFileIsZeroConfig(t *testing.T) {
	dir := t.TempDir()

	cfg, err := LoadModuleConfig(dir)
	if err != nil {
		t.Fatalf("missing jig.toml must not be an error, got: %v", err)
	}
	if cfg.Template.URL != "" || cfg.Build.Action != "" || len(cfg.Renew.Paths) != 0 {
		t.Errorf("expected zero config for missing file, got %+v", cfg)
	}
}

func TestLoadModuleConfig_ParsesAllSections(t *testing.T) {
	dir := t.TempDir()
	content := `
[template]
url = "ssh://git@example.com/templates.git"
ref = "main"
commit = "abc123"

[renew]
paths = ["Gemfile", ".rubocop.yml"]

[build]
action = "allow"
exceptions = ["/spec/**"]
`
	if err := os.WriteFile(filepath.Join(dir, ModuleConfigFileName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadModuleConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Template.URL != "ssh://git@example.com/templates.git" {
		t.Errorf("Template.URL: got %q", cfg.Template.URL)
	}
	if cfg.Template.Ref != "main" || cfg.Template.Commit != "abc123" {
		t.Errorf("Template ref/commit: got %+v", cfg.Template)
	}
	if len(cfg.Renew.Paths) != 2 || cfg.Renew.Paths[0] != "Gemfile" {
		t.Errorf("Renew.Paths: got %v", cfg.Renew.Paths)
	}
	if cfg.Build.Action != BuildActionAllow {
		t.Errorf("Build.Action: got %q", cfg.Build.Action)
	}
	if len(cfg.Build.Exceptions) != 1 || cfg.Build.Exceptions[0] != "/spec/**" {
		t.Errorf("Build.Exceptions: got %v", cfg.Build.Exceptions)
	}
}

func TestLoadModuleConfig_InvalidToml(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ModuleConfigFileName), []byte("this is not toml = ["), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadModuleConfig(dir); err == nil {
		t.Fatal("expected error for invalid toml, got nil")
	}
}

func TestLoadModuleConfig_InvalidBuildAction(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ModuleConfigFileName), []byte("[build]\naction = \"blocklist\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadModuleConfig(dir)
	if err == nil {
		t.Fatal("expected error for invalid build action, got nil")
	}
	if !strings.Contains(err.Error(), "blocklist") {
		t.Errorf("error %q does not name the invalid action", err.Error())
	}
}

func TestModuleConfig_WriteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := ModuleConfig{
		Template: ModuleTemplate{
			URL:    "ssh://git@example.com/templates.git",
			Ref:    "v2",
			Commit: "def456",
		},
		Renew: RenewConfig{Paths: []string{"Rakefile"}},
		Build: BuildConfig{Action: BuildActionDeny, Exceptions: []string{"/extra.txt"}},
	}

	if err := cfg.Write(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := LoadModuleConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.Template != cfg.Template {
		t.Errorf("Template: got %+v, want %+v", loaded.Template, cfg.Template)
	}
	if len(loaded.Renew.Paths) != 1 || loaded.Renew.Paths[0] != "Rakefile" {
		t.Errorf("Renew.Paths: got %v", loaded.Renew.Paths)
	}
	if loaded.Build.Action != cfg.Build.Action || len(loaded.Build.Exceptions) != 1 {
		t.Errorf("Build: got %+v, want %+v", loaded.Build, cfg.Build)
	}
}

func TestBuildConfigValidate(t *testing.T) {
	for _, action := range []string{"", BuildActionAllow, BuildActionDeny} {
		if err := (BuildConfig{Action: action}).Validate(); err != nil {
			t.Errorf("action %q should be valid, got: %v", action, err)
		}
	}
	if err := (BuildConfig{Action: "nope"}).Validate(); err == nil {
		t.Error("expected error for invalid action")
	}
}
