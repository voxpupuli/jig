// SPDX-License-Identifier: GPL-3.0-or-later
package module

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewMetadata(t *testing.T) {
	m := NewMetadata("mymodule", "myuser", "My Name")

	if m.Name != "myuser-mymodule" {
		t.Errorf("Name: got %q, want %q", m.Name, "myuser-mymodule")
	}
	if m.Author != "My Name" {
		t.Errorf("Author: got %q, want %q", m.Author, "My Name")
	}
	if m.Version != "0.1.0" {
		t.Errorf("Version: got %q, want %q", m.Version, "0.1.0")
	}
	if m.License != "Apache-2.0" {
		t.Errorf("License: got %q, want %q", m.License, "Apache-2.0")
	}
	if len(m.Requirements) != 1 || m.Requirements[0].Name != "openvox" {
		t.Errorf("Requirements: expected single puppet requirement, got %v", m.Requirements)
	}
	if m.Dependencies == nil {
		t.Error("Dependencies: expected initialized slice, got nil")
	}
	if m.Tags == nil {
		t.Error("Tags: expected initialized slice, got nil")
	}
}

func TestModuleName(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"standard", "author-module", "module"},
		{"no delimiter", "nodash", "nodash"},
		{"multiple dashes", "author-module-extra", "module-extra"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Metadata{Name: tc.input}
			if got := m.ModuleName(); got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestForgeUsername(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"standard", "author-module", "author"},
		{"no delimiter", "nodash", ""},
		{"multiple dashes", "author-module-extra", "author"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Metadata{Name: tc.input}
			if got := m.ForgeUsername(); got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestReadMetadata(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "metadata.json")

		content := `{
			"name": "author-module",
			"version": "1.2.3",
			"author": "author",
			"license": "MIT",
			"summary": "A test module",
			"source": "",
			"dependencies": [],
			"requirements": [],
			"operatingsystem_support": [],
			"tags": [],
			"pdk-version": "3.4.0"
		}`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		m, err := ReadMetadata(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Name != "author-module" {
			t.Errorf("Name: got %q, want %q", m.Name, "author-module")
		}
		if m.Version != "1.2.3" {
			t.Errorf("Version: got %q, want %q", m.Version, "1.2.3")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := ReadMetadata("/nonexistent/path/metadata.json")
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "metadata.json")
		if err := os.WriteFile(path, []byte("not json {{{"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := ReadMetadata(path)
		if err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})
}

func TestWrite(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "metadata.json")

		original := NewMetadata("mymodule", "myuser", "My Name")
		if err := original.Write(path); err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		read, err := ReadMetadata(path)
		if err != nil {
			t.Fatalf("ReadMetadata failed: %v", err)
		}
		if read.Name != original.Name {
			t.Errorf("Name: got %q, want %q", read.Name, original.Name)
		}
		if read.Version != original.Version {
			t.Errorf("Version: got %q, want %q", read.Version, original.Version)
		}
	})

	t.Run("html not escaped", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "metadata.json")

		m := Metadata{
			Name:         "author-module",
			Source:       "https://github.com/author/module",
			Dependencies: []Dependency{},
			Requirements: []Requirement{},
			Tags:         []string{},
		}
		if err := m.Write(path); err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		// json.Encoder with SetEscapeHTML(false) should leave & < > unescaped
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}
		src, _ := decoded["source"].(string)
		if src != "https://github.com/author/module" {
			t.Errorf("source URL mangled: got %q", src)
		}
	})

	t.Run("unwritable path", func(t *testing.T) {
		err := (&Metadata{}).Write("/nonexistent/dir/metadata.json")
		if err == nil {
			t.Error("expected error for unwritable path, got nil")
		}
	})
}

func TestReadMetadata_PathIsDirectory(t *testing.T) {
	// Passing a directory path instead of a file should error, not panic.
	dir := t.TempDir()
	_, err := ReadMetadata(dir)
	if err == nil {
		t.Error("expected error when path is a directory, got nil")
	}
}

func TestWrite_PathIsDirectory(t *testing.T) {
	// Passing a directory path should error, not silently succeed or panic.
	dir := t.TempDir()
	m := NewMetadata("mymodule", "myuser", "My Name")
	err := m.Write(dir)
	if err == nil {
		t.Error("expected error when write path is a directory, got nil")
	}
}

func TestNewMetadata_EmptyFields(t *testing.T) {
	// Empty forge user or name produces a malformed Name field.
	// These document current behavior -- callers are responsible for
	// validating inputs before calling NewMetadata.
	cases := []struct {
		name       string
		moduleName string
		forgeUser  string
		expected   string
	}{
		{"empty forge user", "mymodule", "", "-mymodule"},
		{"empty module name", "", "myuser", "myuser-"},
		{"both empty", "", "", "-"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMetadata(tc.moduleName, tc.forgeUser, "author")
			if m.Name != tc.expected {
				t.Errorf("Name: got %q, want %q", m.Name, tc.expected)
			}
			// Validate should catch the malformed name
			results := m.Validate()
			warnings := findResults(results, "name", Warning)
			if len(warnings) == 0 {
				t.Errorf("expected Validate to warn about malformed name %q", m.Name)
			}
		})
	}
}

// The template provenance fields must survive a write/read round trip, and
// must be omitted from the JSON entirely when unset so modules that never
// used a remote template keep a clean metadata.json.
func TestTemplateFieldsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")

	m := NewMetadata("mymodule", "myuser", "My Name")
	m.TemplateURL = "ssh://git@git.example.com/templates.git"
	m.TemplateRef = "main"
	m.TemplateCommit = "0123456789abcdef0123456789abcdef01234567"

	if err := m.Write(path); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := ReadMetadata(path)
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if got.TemplateURL != m.TemplateURL {
		t.Errorf("TemplateURL: got %q, want %q", got.TemplateURL, m.TemplateURL)
	}
	if got.TemplateRef != m.TemplateRef {
		t.Errorf("TemplateRef: got %q, want %q", got.TemplateRef, m.TemplateRef)
	}
	if got.TemplateCommit != m.TemplateCommit {
		t.Errorf("TemplateCommit: got %q, want %q", got.TemplateCommit, m.TemplateCommit)
	}
}

func TestTemplateFieldsOmittedWhenUnset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")

	m := NewMetadata("mymodule", "myuser", "My Name")
	if err := m.Write(path); err != nil {
		t.Fatalf("Write: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read metadata.json: %v", err)
	}
	for _, key := range []string{"template-url", "template-ref", "template-commit"} {
		if strings.Contains(string(content), key) {
			t.Errorf("unset %s should be omitted from metadata.json", key)
		}
	}
}
