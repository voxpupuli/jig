// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func makeTransportTemplateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	transportDir := filepath.Join(dir, "transport")
	schemaDir := filepath.Join(transportDir, "schema")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(transportDir, "transport.rb.tmpl"):      "# transport {{.Name}}\n",
		filepath.Join(transportDir, "transport_spec.rb.tmpl"): "# transport spec {{.Name}}\n",
		filepath.Join(transportDir, "device.rb.tmpl"):         "# device {{.Name | pascalCase}}\n",
		filepath.Join(schemaDir, "schema.rb.tmpl"):            "# schema {{.Name | pascalCase}}\n",
		filepath.Join(schemaDir, "schema_spec.rb.tmpl"):       "# schema spec {{.Name}}\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestNewTransport_InvalidNames(t *testing.T) {
	cases := []struct {
		name          string
		transportName string
	}{
		{"empty name", ""},
		{"path separator", "foo/bar"},
		{"backslash", `foo\bar`},
		{"dot dot", ".."},
		{"dot dot slash", "../evil"},
		{"uppercase letter", "MyTransport"},
		{"starts with number", "123transport"},
		{"starts with underscore", "_transport"},
		{"contains double colon", "foo::bar"},
		{"contains hyphen", "my-transport"},
		{"contains space", "my transport"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := makeModuleDir(t, "myuser", "mymodule")
			err := NewTransport(ComponentOptions{
				Name:        tc.transportName,
				WorkDir:     dir,
				TemplateDir: makeTransportTemplateDir(t),
			})
			if err == nil {
				t.Errorf("expected error for transport name %q, got nil", tc.transportName)
			}
			// Verify nothing was written to disk
			if _, statErr := os.Stat(filepath.Join(dir, "lib")); statErr == nil {
				t.Errorf("lib/ should not have been created for invalid name %q", tc.transportName)
			}
		})
	}
}

func TestNewTransport_RejectsInvalidModuleDirectory(t *testing.T) {
	dir := t.TempDir()
	err := NewTransport(ComponentOptions{
		Name:        "mytransport",
		WorkDir:     dir,
		TemplateDir: makeTransportTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for missing metadata.json, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "lib")); statErr == nil {
		t.Error("lib/ should not have been created in an invalid module directory")
	}
}

func TestNewTransport_RejectsCorruptMetadata(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte("not json {{{"), 0644); err != nil {
		t.Fatal(err)
	}
	err := NewTransport(ComponentOptions{
		Name:        "mytransport",
		WorkDir:     dir,
		TemplateDir: makeTransportTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for corrupt metadata.json, got nil")
	}
}

func TestNewTransport_RejectsExistingFiles(t *testing.T) {
	cases := []struct {
		name     string
		existing func(dir string) string
	}{
		{
			"transport file already exists",
			func(dir string) string {
				p := filepath.Join(dir, "lib", "puppet", "transport", "mytransport.rb")
				mustWriteFile(t, p, "# existing")
				return p
			},
		},
		{
			"transport spec file already exists",
			func(dir string) string {
				p := filepath.Join(dir, "spec", "unit", "puppet", "transport", "mytransport_spec.rb")
				mustWriteFile(t, p, "# existing")
				return p
			},
		},
		{
			"schema file already exists",
			func(dir string) string {
				p := filepath.Join(dir, "lib", "puppet", "transport", "schema", "mytransport.rb")
				mustWriteFile(t, p, "# existing")
				return p
			},
		},
		{
			"schema spec file already exists",
			func(dir string) string {
				p := filepath.Join(dir, "spec", "unit", "puppet", "transport", "schema", "mytransport_spec.rb")
				mustWriteFile(t, p, "# existing")
				return p
			},
		},
		{
			"device file already exists",
			func(dir string) string {
				p := filepath.Join(dir, "lib", "puppet", "util", "network_device", "mytransport", "device.rb")
				mustWriteFile(t, p, "# existing")
				return p
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := makeModuleDir(t, "myuser", "mymodule")
			existing := tc.existing(dir)
			err := NewTransport(ComponentOptions{
				Name:        "mytransport",
				WorkDir:     dir,
				TemplateDir: makeTransportTemplateDir(t),
			})
			if err == nil {
				t.Errorf("expected error when %s, got nil", tc.name)
			}
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

func TestNewTransport_NoPartialStateOnTemplateFailure(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")

	tmplDir := t.TempDir()
	transportDir := filepath.Join(tmplDir, "transport")
	schemaDir := filepath.Join(transportDir, "schema")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(transportDir, "transport.rb.tmpl"):      "{{.Name}}\n",
		filepath.Join(transportDir, "transport_spec.rb.tmpl"): "{{.Name}}\n",
		filepath.Join(transportDir, "device.rb.tmpl"):         "{{invalid template\n",
		filepath.Join(schemaDir, "schema.rb"):                 "{{.Name}}\n",
		filepath.Join(schemaDir, "schema_spec.rb"):            "{{.Name}}\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	err := NewTransport(ComponentOptions{
		Name:        "mytransport",
		WorkDir:     dir,
		TemplateDir: tmplDir,
	})
	if err == nil {
		t.Error("expected error from broken template, got nil")
	}
}

// mustWriteFile is a test helper that creates a file and all necessary parent
// directories, fatally failing the test on any error.
func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create directories for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}
