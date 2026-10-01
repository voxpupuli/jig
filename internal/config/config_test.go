// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// quietLogger returns a logger that discards output so tests stay silent.
func quietLogger() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

// writeConfig writes the given TOML into a temp file and returns its path.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	return path
}

// With no config file present, the runner must fall back to the documented
// defaults (local runner, docker engine, voxbox image).
func TestLoad_RunnerDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.toml")

	cfg, err := Load(path, quietLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Runner.Type != DefaultRunnerType {
		t.Errorf("runner type: got %q, want %q", cfg.Runner.Type, DefaultRunnerType)
	}
	if cfg.Runner.Engine != DefaultRunnerEngine {
		t.Errorf("runner engine: got %q, want %q", cfg.Runner.Engine, DefaultRunnerEngine)
	}
	if cfg.Runner.Image != DefaultRunnerImage {
		t.Errorf("runner image: got %q, want %q", cfg.Runner.Image, DefaultRunnerImage)
	}
}

// Values in the config file must populate the nested runner struct.
func TestLoad_RunnerFromFile(t *testing.T) {
	path := writeConfig(t, `
forge_username = "jdoe"

[runner]
type   = "voxbox"
engine = "podman"
image  = "localhost/voxbox:dev"
`)

	cfg, err := Load(path, quietLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ForgeUsername != "jdoe" {
		t.Errorf("forge_username: got %q, want %q", cfg.ForgeUsername, "jdoe")
	}
	if cfg.Runner.Type != "voxbox" {
		t.Errorf("runner type: got %q, want %q", cfg.Runner.Type, "voxbox")
	}
	if cfg.Runner.Engine != "podman" {
		t.Errorf("runner engine: got %q, want %q", cfg.Runner.Engine, "podman")
	}
	if cfg.Runner.Image != "localhost/voxbox:dev" {
		t.Errorf("runner image: got %q, want %q", cfg.Runner.Image, "localhost/voxbox:dev")
	}
}

// ssh_accept_new must default to false (always ask about unknown host keys)
// and be settable from the config file or the JIG_SSH_ACCEPT_NEW env var.
// It lives in per-user config only -- never metadata.json -- because it is a
// trust decision.
func TestLoad_SSHAcceptNew(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.toml")
	cfg, err := Load(missing, quietLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SSHAcceptNew {
		t.Error("ssh_accept_new must default to false")
	}

	path := writeConfig(t, "ssh_accept_new = true\n")
	cfg, err = Load(path, quietLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.SSHAcceptNew {
		t.Error("ssh_accept_new from the config file was not applied")
	}
}

func TestLoad_SSHAcceptNewEnvOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.toml")
	t.Setenv("JIG_SSH_ACCEPT_NEW", "true")

	cfg, err := Load(path, quietLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.SSHAcceptNew {
		t.Error("JIG_SSH_ACCEPT_NEW env override was not applied")
	}
}

// An environment variable must override the config file for nested runner keys
// (JIG_RUNNER_TYPE -> runner.type). This is the override path ananace's users
// rely on without a CLI flag.
func TestLoad_RunnerEnvOverridesFile(t *testing.T) {
	path := writeConfig(t, `
[runner]
type = "local"
`)

	t.Setenv("JIG_RUNNER_TYPE", "voxbox")

	cfg, err := Load(path, quietLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Runner.Type != "voxbox" {
		t.Errorf("runner type: env override not applied, got %q, want %q", cfg.Runner.Type, "voxbox")
	}
}

// The env override must also work when the key is absent from the file, since
// AutomaticEnv only reaches Unmarshal because the defaults register the keys.
func TestLoad_RunnerEnvOverridesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.toml")

	t.Setenv("JIG_RUNNER_ENGINE", "podman")

	cfg, err := Load(path, quietLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Runner.Engine != "podman" {
		t.Errorf("runner engine: env override not applied, got %q, want %q", cfg.Runner.Engine, "podman")
	}
}

// [template.vars] keeps its key case (viper would lowercase it) and yields
// the layers for one module, most specific first.
func TestGlobalTemplateVars_Layers(t *testing.T) {
	path := writeConfig(t, `
[template.vars.default]
enable_junit = false
ci = "github"

[template.vars.acme]
enable_junit = true

[template.vars.acme.widget]
ci = "gitlab"

[template.vars.other]
ci = "jenkins"
`)
	cfg, err := Load(path, quietLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := []struct {
		name        string
		author, mod string
		wantLayers  []map[string]any
	}{
		{
			name:   "author and module sections",
			author: "acme", mod: "widget",
			wantLayers: []map[string]any{
				{"ci": "gitlab"},
				{"enable_junit": true},
				{"enable_junit": false, "ci": "github"},
			},
		},
		{
			name:   "author section only",
			author: "acme", mod: "gadget",
			wantLayers: []map[string]any{
				{"enable_junit": true},
				{"enable_junit": false, "ci": "github"},
			},
		},
		{
			name:   "defaults only",
			author: "nobody", mod: "widget",
			wantLayers: []map[string]any{
				{"enable_junit": false, "ci": "github"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			layers, err := cfg.GlobalTemplateVars(tc.author, tc.mod)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(layers, tc.wantLayers) {
				t.Errorf("layers: got %v, want %v", layers, tc.wantLayers)
			}
		})
	}
}

// "default" is reserved: it always means the defaults section, even for a
// module whose author is literally named default.
func TestGlobalTemplateVars_DefaultIsNeverAnAuthor(t *testing.T) {
	path := writeConfig(t, `
[template.vars.default]
ci = "github"
`)
	cfg, err := Load(path, quietLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	layers, err := cfg.GlobalTemplateVars("default", "widget")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layers) != 1 {
		t.Errorf("default must only be read as the defaults section, got layers %v", layers)
	}
}

func TestGlobalTemplateVars_RejectsInvalidSections(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"camelCase name", "[template.vars.default]\nenableJunit = true\n", "enableJunit"},
		{"table under default", "[template.vars.default.fixtures]\nrepo = \"x\"\n", "template.vars.default.fixtures"},
		{"table under module", "[template.vars.acme.widget.fixtures]\nrepo = \"x\"\n", "template.vars.acme.widget.fixtures"},
		{"bare variable", "[template.vars]\nci = \"x\"\n", "template.vars.default"},
		{"camelCase in another author", "[template.vars.other]\nCI = \"x\"\n", "CI"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := Load(writeConfig(t, tc.body), quietLogger())
			if err != nil {
				t.Fatalf("unexpected load error: %v", err)
			}
			_, err = cfg.GlobalTemplateVars("acme", "widget")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected an error mentioning %q, got: %v", tc.want, err)
			}
		})
	}
}

// A non-TOML config passed with --config must keep loading through viper,
// template variables included, rather than being parsed as TOML.
func TestLoad_YAMLConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	body := "forge_username: jdoe\ntemplate:\n  vars:\n    default:\n      ci: gitlab\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path, quietLogger())
	if err != nil {
		t.Fatalf("a YAML config must still load, got: %v", err)
	}
	if cfg.ForgeUsername != "jdoe" {
		t.Errorf("forge_username: got %q", cfg.ForgeUsername)
	}
	layers, err := cfg.GlobalTemplateVars("acme", "widget")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layers) != 1 || layers[0]["ci"] != "gitlab" {
		t.Errorf("template vars from YAML: got %v", layers)
	}
}
