// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/voxpupuli/jig/v2/internal/config"
	"github.com/voxpupuli/jig/v2/internal/template"
)

const hieraManifest = `
[vars.enable_test_hiera]
type    = "bool"
default = false

[vars.ci]
type    = "string"
default = "github"

[[files]]
path = "spec/fixtures/hiera/**"
when = ".Vars.enable_test_hiera"
`

// varsTemplateDir builds a template dir with a jig-template.toml and a
// conditional Hiera fixture tree holding a verbatim file and an empty
// directory, plus a README that uses a variable.
func varsTemplateDir(t *testing.T) (string, template.Manifest) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, template.ManifestFileName), []byte(hieraManifest), 0644); err != nil {
		t.Fatal(err)
	}
	writeModuleTemplate(t, dir, "README.md.tmpl", "# {{.ModuleName}} ci={{.Vars.ci}}{{ if .Vars.enable_test_hiera }} hiera{{ end }}\n")
	writeModuleTemplate(t, dir, "spec/fixtures/hiera/hiera.yaml", "---\nversion: 5\n")
	writeModuleTemplate(t, dir, "spec/fixtures/hiera/data/nodes/.gitkeep", "")

	m, err := template.LoadManifest(dir)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	return dir, m
}

func TestNewModule_WhenRulesAndVars(t *testing.T) {
	cases := []struct {
		name       string
		hiera      bool
		wantReadme string
	}{
		{"hiera off", false, "# mymodule ci=gitlab\n"},
		{"hiera on", true, "# mymodule ci=gitlab hiera\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmplDir, manifest := varsTemplateDir(t)
			vars := map[string]any{"enable_test_hiera": tc.hiera, "ci": "gitlab"}
			target := t.TempDir()

			err := NewModule(Options{
				ForgeUser:   "myuser",
				Name:        "mymodule",
				Author:      "Me",
				License:     "Apache-2.0",
				TargetDir:   target,
				TemplateDir: tmplDir,
				Manifest:    manifest,
				Vars:        vars,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			moduleDir := filepath.Join(target, "mymodule")

			if got := readModuleFile(t, moduleDir, "README.md"); got != tc.wantReadme {
				t.Errorf("README.md: got %q, want %q", got, tc.wantReadme)
			}
			for _, f := range []string{"spec/fixtures/hiera/hiera.yaml", "spec/fixtures/hiera/data/nodes/.gitkeep"} {
				_, statErr := os.Stat(filepath.Join(moduleDir, filepath.FromSlash(f)))
				if exists := statErr == nil; exists != tc.hiera {
					t.Errorf("%s: exists=%v, want %v", f, exists, tc.hiera)
				}
			}

			cfg, err := config.LoadModuleConfig(moduleDir)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(cfg.Template.Vars, vars) {
				t.Errorf("jig.toml [template.vars]: got %v, want %v", cfg.Template.Vars, vars)
			}
		})
	}
}

// A file the template stopped generating must be reported by renew and
// left in place, never deleted.
func TestRenew_ReportsFilesNoLongerGenerated(t *testing.T) {
	tmplDir, manifest := varsTemplateDir(t)
	moduleDir := makeModuleDir(t, "myuser", "mymodule")
	writeModuleFile(t, moduleDir, "spec/fixtures/hiera/hiera.yaml", "old hiera\n")
	writeModuleFile(t, moduleDir, "README.md", "# old\n")

	var out strings.Builder
	err := Renew(RenewOptions{
		ModuleDir:   moduleDir,
		TemplateDir: tmplDir,
		Paths:       []string{"README.md", "spec/**"},
		Out:         &out,
		Manifest:    manifest,
		Vars:        map[string]any{"enable_test_hiera": false, "ci": "circle"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := readModuleFile(t, moduleDir, "spec/fixtures/hiera/hiera.yaml"); got != "old hiera\n" {
		t.Errorf("a file no longer generated must be left alone, got %q", got)
	}
	if !strings.Contains(out.String(), "no longer generates") || !strings.Contains(out.String(), "spec/fixtures/hiera/hiera.yaml") {
		t.Errorf("output should report the file no longer generated, got: %q", out.String())
	}
	if strings.Contains(out.String(), ".gitkeep") {
		t.Errorf("files the module never had must not be reported, got: %q", out.String())
	}
	if got := readModuleFile(t, moduleDir, "README.md"); got != "# mymodule ci=circle\n" {
		t.Errorf("README.md must render with the given vars, got %q", got)
	}
}

func TestRenew_WhenTrueCreatesFiles(t *testing.T) {
	tmplDir, manifest := varsTemplateDir(t)
	moduleDir := makeModuleDir(t, "myuser", "mymodule")

	err := Renew(RenewOptions{
		ModuleDir:   moduleDir,
		TemplateDir: tmplDir,
		Paths:       []string{"spec/**"},
		Out:         &strings.Builder{},
		Manifest:    manifest,
		Vars:        map[string]any{"enable_test_hiera": true, "ci": "github"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := readModuleFile(t, moduleDir, "spec/fixtures/hiera/hiera.yaml"); got != "---\nversion: 5\n" {
		t.Errorf("hiera.yaml: got %q", got)
	}
}

func TestNewClass_RendersVars(t *testing.T) {
	tmplDir := t.TempDir()
	classDir := filepath.Join(tmplDir, "class")
	if err := os.MkdirAll(classDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(classDir, "class.pp.tmpl"), []byte("class {{.Name}} { # {{.Vars.ci}}\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(classDir, "class_spec.rb.tmpl"), []byte("# {{.Name}}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	moduleDir := makeModuleDir(t, "myuser", "mymodule")

	err := NewClass(ComponentOptions{Name: "foo", TemplateDir: tmplDir, WorkDir: moduleDir, Vars: map[string]any{"ci": "gitlab"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := readModuleFile(t, moduleDir, "manifests/foo.pp"); got != "class mymodule::foo { # gitlab\n}\n" {
		t.Errorf("manifests/foo.pp: got %q", got)
	}
}

func TestConvertModule_RecordsTemplateVars(t *testing.T) {
	manifest := template.Manifest{Vars: map[string]template.VarSpec{
		"ci":    {Type: template.VarTypeString, Default: "github"},
		"junit": {Type: template.VarTypeBool, Default: false},
		"hiera": {Type: template.VarTypeBool, Default: false},
	}}
	globals := func(author, module string) ([]map[string]any, error) {
		if author != "myuser" || module != "mymodule" {
			t.Errorf("user config looked up for %s/%s, want myuser/mymodule", author, module)
		}
		return []map[string]any{{"ci": "gitlab", "junit": true, "hiera": true}}, nil
	}

	cases := []struct {
		name     string
		existing string
		flags    map[string]any
		want     map[string]any
	}{
		{
			name: "new jig.toml gets user config values",
			want: map[string]any{"ci": "gitlab", "junit": true, "hiera": true},
		},
		{
			name:     "existing jig.toml values win over the user config",
			existing: "[template.vars]\nci = \"circle\"\n",
			want:     map[string]any{"ci": "circle", "junit": true, "hiera": true},
		},
		{
			name:     "flags win over everything",
			existing: "[template.vars]\nci = \"circle\"\n",
			flags:    map[string]any{"ci": "drone", "hiera": false},
			want:     map[string]any{"ci": "drone", "junit": true, "hiera": false},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			moduleDir := makeModuleDir(t, "myuser", "mymodule")
			if tc.existing != "" {
				writeModuleFile(t, moduleDir, config.ModuleConfigFileName, tc.existing)
			}

			err := ConvertModule(ConvertOptions{
				TargetDir:   moduleDir,
				Out:         &strings.Builder{},
				TemplateURL: "https://example.com/templates.git",
				Manifest:    manifest,
				FlagVars:    tc.flags,
				GlobalVars:  globals,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cfg, err := config.LoadModuleConfig(moduleDir)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(cfg.Template.Vars, tc.want) {
				t.Errorf("[template.vars]: got %v, want %v", cfg.Template.Vars, tc.want)
			}
			if cfg.Template.URL != "https://example.com/templates.git" {
				t.Errorf("[template] url not recorded: %+v", cfg.Template)
			}
			if cfg.Template.Commit != "" {
				t.Errorf("convert renders nothing from the template, so it must not record a commit, got %q", cfg.Template.Commit)
			}
		})
	}
}

// Without a template source or variables, an existing jig.toml must not be
// rewritten (rewriting drops its comments).
func TestConvertModule_LeavesUnchangedJigTomlAlone(t *testing.T) {
	moduleDir := makeModuleDir(t, "myuser", "mymodule")
	original := "# hand-written comment\n[renew]\npaths = [\"Gemfile\"]\n"
	writeModuleFile(t, moduleDir, config.ModuleConfigFileName, original)

	var out strings.Builder
	if err := ConvertModule(ConvertOptions{TargetDir: moduleDir, Out: &out}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := readModuleFile(t, moduleDir, config.ModuleConfigFileName); got != original {
		t.Errorf("jig.toml must be untouched, got %q", got)
	}
}

// A list variable reads back from jig.toml as []any but resolves to
// []string; convert must still see "nothing changed" and leave the file
// (and its comments) alone on a second run.
func TestConvertModule_IdempotentWithListVar(t *testing.T) {
	manifest := template.Manifest{Vars: map[string]template.VarSpec{
		"fixtures": {Type: template.VarTypeList, Default: []string{"a/b"}},
		"n":        {Type: template.VarTypeInt, Default: int64(2)},
	}}
	moduleDir := makeModuleDir(t, "myuser", "mymodule")
	opts := ConvertOptions{TargetDir: moduleDir, Out: &strings.Builder{}, Manifest: manifest}
	if err := ConvertModule(opts); err != nil {
		t.Fatalf("first convert: %v", err)
	}
	path := filepath.Join(moduleDir, config.ModuleConfigFileName)
	withComment := "# keep me\n" + readModuleFile(t, moduleDir, config.ModuleConfigFileName)
	if err := os.WriteFile(path, []byte(withComment), 0644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	opts.Out = &out
	if err := ConvertModule(opts); err != nil {
		t.Fatalf("second convert: %v", err)
	}
	if got := readModuleFile(t, moduleDir, config.ModuleConfigFileName); got != withComment {
		t.Errorf("an unchanged jig.toml must not be rewritten, got %q", got)
	}
	if strings.Contains(out.String(), "updated") {
		t.Errorf("output must not report an update, got %q", out.String())
	}
}

// Changing the recorded url clears the old commit; keeping it keeps it.
func TestConvertModule_TemplateURLAndCommit(t *testing.T) {
	cases := []struct {
		name       string
		url        string
		wantURL    string
		wantCommit string
	}{
		{"no url flag keeps everything", "", "https://old.example/t.git", "old123"},
		{"same url keeps the commit", "https://old.example/t.git", "https://old.example/t.git", "old123"},
		{"new url clears the commit", "https://new.example/t.git", "https://new.example/t.git", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			moduleDir := makeModuleDir(t, "myuser", "mymodule")
			writeModuleFile(t, moduleDir, config.ModuleConfigFileName, "[template]\nurl = \"https://old.example/t.git\"\ncommit = \"old123\"\n")
			if err := ConvertModule(ConvertOptions{TargetDir: moduleDir, Out: &strings.Builder{}, TemplateURL: tc.url}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			cfg, err := config.LoadModuleConfig(moduleDir)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Template.URL != tc.wantURL || cfg.Template.Commit != tc.wantCommit {
				t.Errorf("got url=%q commit=%q, want url=%q commit=%q", cfg.Template.URL, cfg.Template.Commit, tc.wantURL, tc.wantCommit)
			}
		})
	}
}
