// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func makeProviderTemplateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	providerDir := filepath.Join(dir, "provider")
	if err := os.MkdirAll(providerDir, 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"provider.rb.tmpl":      "# provider {{.Name}}\n",
		"provider_spec.rb.tmpl": "# provider spec {{.Name}}\n",
		"type.rb.tmpl":          "# type {{.Name | upperFirst}}\n",
		"type_spec.rb.tmpl":     "# type spec {{.Name}}\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(providerDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestNewProvider_InvalidNames(t *testing.T) {
	cases := []struct {
		name         string
		providerName string
	}{
		{"empty name", ""},
		{"path separator", "foo/bar"},
		{"backslash", `foo\bar`},
		{"dot dot", ".."},
		{"dot dot slash", "../evil"},
		{"uppercase letter", "MyProvider"},
		{"starts with number", "123provider"},
		{"starts with underscore", "_provider"},
		{"contains double colon", "foo::bar"},
		{"contains hyphen", "my-provider"},
		{"contains space", "my provider"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := makeModuleDir(t, "myuser", "mymodule")
			err := NewProvider(ComponentOptions{
				Name:        tc.providerName,
				WorkDir:     dir,
				TemplateDir: makeProviderTemplateDir(t),
			})
			if err == nil {
				t.Errorf("expected error for provider name %q, got nil", tc.providerName)
			}
			// Verify nothing was written to disk
			entries, readErr := os.ReadDir(filepath.Join(dir, "lib"))
			if readErr == nil && len(entries) > 0 {
				t.Errorf("expected no files written for invalid name %q, but lib/ was created", tc.providerName)
			}
		})
	}
}

func TestNewProvider_RejectsInvalidModuleDirectory(t *testing.T) {
	dir := t.TempDir()
	err := NewProvider(ComponentOptions{
		Name:        "myprovider",
		WorkDir:     dir,
		TemplateDir: makeProviderTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for missing metadata.json, got nil")
	}
	// Verify no partial state was written
	if _, statErr := os.Stat(filepath.Join(dir, "lib")); statErr == nil {
		t.Error("lib/ directory should not have been created in an invalid module directory")
	}
}

func TestNewProvider_RejectsCorruptMetadata(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte("not json {{{"), 0644); err != nil {
		t.Fatal(err)
	}
	err := NewProvider(ComponentOptions{
		Name:        "myprovider",
		WorkDir:     dir,
		TemplateDir: makeProviderTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for corrupt metadata.json, got nil")
	}
}

func TestNewProvider_RejectsExistingFiles(t *testing.T) {
	cases := []struct {
		name     string
		existing func(dir string) string
	}{
		{
			"provider file already exists",
			func(dir string) string {
				p := filepath.Join(dir, "lib", "puppet", "provider", "myprovider", "myprovider.rb")
				if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
					panic(err)
				}
				if err := os.WriteFile(p, []byte("# existing"), 0644); err != nil {
					panic(err)
				}
				return p
			},
		},
		{
			"provider spec file already exists",
			func(dir string) string {
				p := filepath.Join(dir, "spec", "unit", "puppet", "provider", "myprovider", "myprovider_spec.rb")
				if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
					panic(err)
				}
				if err := os.WriteFile(p, []byte("# existing"), 0644); err != nil {
					panic(err)
				}
				return p
			},
		},
		{
			"type file already exists",
			func(dir string) string {
				p := filepath.Join(dir, "lib", "puppet", "type", "myprovider.rb")
				if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
					panic(err)
				}
				if err := os.WriteFile(p, []byte("# existing"), 0644); err != nil {
					panic(err)
				}
				return p
			},
		},
		{
			"type spec file already exists",
			func(dir string) string {
				p := filepath.Join(dir, "spec", "unit", "puppet", "type", "myprovider_spec.rb")
				if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
					panic(err)
				}
				if err := os.WriteFile(p, []byte("# existing"), 0644); err != nil {
					panic(err)
				}
				return p
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := makeModuleDir(t, "myuser", "mymodule")
			existing := tc.existing(dir)
			err := NewProvider(ComponentOptions{
				Name:        "myprovider",
				WorkDir:     dir,
				TemplateDir: makeProviderTemplateDir(t),
			})
			if err == nil {
				t.Errorf("expected error when %s, got nil", tc.name)
			}
			// The pre-existing file must not have been overwritten
			content, readErr := os.ReadFile(existing)
			if readErr != nil {
				t.Fatalf("could not read pre-existing file: %v", readErr)
			}
			if string(content) != "# existing" {
				t.Errorf("pre-existing file was overwritten")
			}
		})
	}
}

func TestNewProvider_NoPartialStateOnTemplateFailure(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")

	// Template dir with a broken template to force a render failure
	tmplDir := t.TempDir()
	providerDir := filepath.Join(tmplDir, "provider")
	if err := os.MkdirAll(providerDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(providerDir, "provider.rb.tmpl"), []byte("{{.Name}}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(providerDir, "provider_spec.rb.tmpl"), []byte("{{.Name}}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(providerDir, "type.rb.tmpl"), []byte("{{invalid template\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(providerDir, "type_spec.rb.tmpl"), []byte("{{.Name}}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := NewProvider(ComponentOptions{
		Name:        "myprovider",
		WorkDir:     dir,
		TemplateDir: tmplDir,
	})
	if err == nil {
		t.Error("expected error from broken template, got nil")
	}
}
