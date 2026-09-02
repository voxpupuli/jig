// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/voxpupuli/jig/v2/internal/module"
)

func TestSplitForgeName(t *testing.T) {
	cases := []struct {
		raw        string
		forgeUser  string
		moduleName string
	}{
		{"puppet-nftables", "puppet", "nftables"},
		{"binford2k-demo", "binford2k", "demo"},
		// Puppet's own Modulefile accepted both "user-mod" and "user/mod".
		{"binford2k/demo", "binford2k", "demo"},
		{"nodashormodname", "", "nodashormodname"},
	}

	for _, c := range cases {
		forgeUser, name := SplitForgeName(c.raw)
		if forgeUser != c.forgeUser || name != c.moduleName {
			t.Errorf("SplitForgeName(%q) = (%q, %q), want (%q, %q)", c.raw, forgeUser, name, c.forgeUser, c.moduleName)
		}
	}
}

func writeModulefile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "Modulefile")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write Modulefile: %v", err)
	}
	return path
}

func TestParseModulefile(t *testing.T) {
	path := writeModulefile(t, `name    'binford2k-demo'
version '2.3.4'
author  'Binford Tools'
license 'MIT'
summary 'A demo module'
description 'Longer description text'
source 'https://example.com/demo'
project_page 'https://example.com/demo/docs'
dependency 'puppetlabs/stdlib', '>= 3.2.0 < 5.0.0'
`)

	data, err := ParseModulefile(path)
	if err != nil {
		t.Fatalf("ParseModulefile failed: %v", err)
	}

	want := ModulefileData{
		Name:        "binford2k-demo",
		Version:     "2.3.4",
		Author:      "Binford Tools",
		License:     "MIT",
		Summary:     "A demo module",
		Description: "Longer description text",
		Source:      "https://example.com/demo",
		ProjectPage: "https://example.com/demo/docs",
		Dependencies: []module.Dependency{
			// puppetlabs/stdlib is normalized to the current puppetlabs-stdlib
			// naming convention.
			{Name: "puppetlabs-stdlib", VersionRequirement: ">= 3.2.0 < 5.0.0"},
		},
	}
	if !reflect.DeepEqual(data, want) {
		t.Errorf("ParseModulefile:\n got  %+v\n want %+v", data, want)
	}
}

// A slash-separated dependency/'user/mod'/ name is the legacy Modulefile
// form; jig should normalize it to the current dash convention.
func TestParseModulefile_DependencyNameNormalized(t *testing.T) {
	path := writeModulefile(t, "dependency 'puppetlabs/stdlib', '>= 4.0.0'\n")

	data, err := ParseModulefile(path)
	if err != nil {
		t.Fatalf("ParseModulefile failed: %v", err)
	}
	if len(data.Dependencies) != 1 || data.Dependencies[0].Name != "puppetlabs-stdlib" {
		t.Errorf("expected dependency name normalized to puppetlabs-stdlib, got %+v", data.Dependencies)
	}
}

// Modulefile commonly wrapped a dependency call across two lines, with the
// name argument's trailing comma as the continuation signal.
func TestParseModulefile_MultilineDependency(t *testing.T) {
	path := writeModulefile(t, "dependency 'puppetlabs/stdlib',\n           '>= 4.0.0'\n")

	data, err := ParseModulefile(path)
	if err != nil {
		t.Fatalf("ParseModulefile failed: %v", err)
	}
	if len(data.Dependencies) != 1 {
		t.Fatalf("expected one dependency, got %+v", data.Dependencies)
	}
	dep := data.Dependencies[0]
	if dep.Name != "puppetlabs-stdlib" || dep.VersionRequirement != ">= 4.0.0" {
		t.Errorf("expected puppetlabs-stdlib >= 4.0.0, got %+v", dep)
	}
}
