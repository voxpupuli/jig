// SPDX-License-Identifier: GPL-3.0-or-later
package template

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

type whenData struct {
	ModuleName string
	Vars       map[string]any
}

func TestLoadManifest_Absent(t *testing.T) {
	for _, dir := range []string{"", t.TempDir()} {
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("dir %q: a missing manifest must not be an error, got: %v", dir, err)
		}
		if len(m.Vars) != 0 || len(m.Files) != 0 {
			t.Errorf("dir %q: expected an empty manifest, got %+v", dir, m)
		}
	}
}

func TestLoadManifest_Valid(t *testing.T) {
	dir := writeManifest(t, `
[vars.enable_test_hiera]
type        = "bool"
default     = true
description = "Generate a Hiera fixture tree"
prompt      = true

[vars.fixtures]
type = "list"
default = ["a/b"]

[vars.ci]
type = "string"

[[files]]
path = "spec/fixtures/hiera/**"
when = ".Vars.enable_test_hiera"
`)
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Vars["enable_test_hiera"].Prompt || m.Vars["enable_test_hiera"].Default != true {
		t.Errorf("enable_test_hiera: got %+v", m.Vars["enable_test_hiera"])
	}
	if !reflect.DeepEqual(m.Vars["fixtures"].Default, []string{"a/b"}) {
		t.Errorf("list defaults must be normalized to []string, got %#v", m.Vars["fixtures"].Default)
	}
	if m.Vars["ci"].Default != "" {
		t.Errorf("a variable without a default must default to its type's zero value, got %#v", m.Vars["ci"].Default)
	}
	if len(m.Files) != 1 {
		t.Fatalf("expected one file rule, got %d", len(m.Files))
	}
}

func TestLoadManifest_Invalid(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"unknown field", "[vars.x]\ntype = \"bool\"\ndefualt = true\n", "defualt"},
		{"missing type", "[vars.x]\ndefault = true\n", "type is required"},
		{"unknown type", "[vars.x]\ntype = \"float\"\n", "unknown type"},
		{"bad name", "[vars.enableJunit]\ntype = \"bool\"\n", "enableJunit"},
		{"default type mismatch", "[vars.x]\ntype = \"bool\"\ndefault = \"yes\"\n", "default"},
		{"list default with non-strings", "[vars.x]\ntype = \"list\"\ndefault = [1]\n", "default"},
		{"file without when", "[[files]]\npath = \"a\"\n", "when is required"},
		{"file without path", "[[files]]\nwhen = \"true\"\n", "path is required"},
		{"negated path", "[vars.x]\ntype = \"bool\"\n[[files]]\npath = \"!a\"\nwhen = \".Vars.x\"\n", "negated"},
		{"when syntax error", "[vars.x]\ntype = \"bool\"\n[[files]]\npath = \"a\"\nwhen = \"and (.Vars.x\"\n", "invalid when"},
		{"when uses undeclared var", "[vars.x]\ntype = \"bool\"\n[[files]]\npath = \"a\"\nwhen = \".Vars.y\"\n", ".Vars.y"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadManifest(writeManifest(t, tc.content))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q, got: %v", tc.want, err)
			}
		})
	}
}

func TestManifestIncludes(t *testing.T) {
	m, err := LoadManifest(writeManifest(t, `
[vars.hiera]
type = "bool"

[vars.ci]
type = "string"

[[files]]
path = "spec/fixtures/hiera/**"
when = ".Vars.hiera"

[[files]]
path = ".gitlab-ci.yml"
when = 'eq .Vars.ci "gitlab"'
`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := []struct {
		path string
		vars map[string]any
		want bool
	}{
		{"spec/fixtures/hiera/hiera.yaml", map[string]any{"hiera": true, "ci": ""}, true},
		{"spec/fixtures/hiera/hiera.yaml", map[string]any{"hiera": false, "ci": ""}, false},
		{"spec/fixtures/hiera/data/nodes/.gitkeep", map[string]any{"hiera": false, "ci": ""}, false},
		{"spec/spec_helper.rb", map[string]any{"hiera": false, "ci": ""}, true},
		{".gitlab-ci.yml", map[string]any{"hiera": false, "ci": "gitlab"}, true},
		{".gitlab-ci.yml", map[string]any{"hiera": false, "ci": "github"}, false},
	}
	for _, tc := range cases {
		got, err := m.Includes(tc.path, whenData{ModuleName: "m", Vars: tc.vars})
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.path, err)
		}
		if got != tc.want {
			t.Errorf("%s with %v: got %v, want %v", tc.path, tc.vars, got, tc.want)
		}
	}
}

// A when expression must produce true or false; anything else (a string
// variable used directly, say) is an error rather than silently truthy.
func TestManifestIncludes_NonBooleanIsError(t *testing.T) {
	m, err := LoadManifest(writeManifest(t, `
[vars.ci]
type = "string"

[[files]]
path = "ci.yml"
when = ".Vars.ci"
`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = m.Includes("ci.yml", whenData{Vars: map[string]any{"ci": "gitlab"}})
	if err == nil || !strings.Contains(err.Error(), "true or false") {
		t.Errorf("expected a true-or-false error, got: %v", err)
	}
}
