// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// --- GetMetadata ---

func TestGetMetadata(t *testing.T) {
	t.Run("valid module directory", func(t *testing.T) {
		dir := makeModuleDir(t, "myuser", "mymodule")
		m, err := GetMetadata(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Name != "myuser-mymodule" {
			t.Errorf("Name: got %q, want %q", m.Name, "myuser-mymodule")
		}
	})

	t.Run("missing metadata.json", func(t *testing.T) {
		dir := t.TempDir()
		_, err := GetMetadata(dir)
		if err == nil {
			t.Error("expected error for missing metadata.json, got nil")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte("{{{"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := GetMetadata(dir)
		if err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})
}

// --- RenderTemplates ---

func TestRenderTemplates(t *testing.T) {
	t.Run("renders files to disk", func(t *testing.T) {
		dir := t.TempDir()
		renderer := &fakeRenderer{output: "rendered content"}
		templates := []TemplateFile{
			{FileName: "foo.pp", Destination: filepath.Join(dir, "manifests", "foo.pp")},
			{FileName: "bar.pp", Destination: filepath.Join(dir, "manifests", "bar.pp")},
		}

		if err := RenderTemplates(renderer, templates, nil, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, tf := range templates {
			content, err := os.ReadFile(tf.Destination)
			if err != nil {
				t.Errorf("expected file %s to exist: %v", tf.Destination, err)
				continue
			}
			if string(content) != "rendered content" {
				t.Errorf("file %s: got %q, want %q", tf.Destination, string(content), "rendered content")
			}
		}
	})

	t.Run("returns error when file exists and overwrite is false", func(t *testing.T) {
		dir := t.TempDir()
		existing := filepath.Join(dir, "init.pp")
		if err := os.WriteFile(existing, []byte("original"), 0644); err != nil {
			t.Fatal(err)
		}

		renderer := &fakeRenderer{output: "new content"}
		templates := []TemplateFile{
			{FileName: "init.pp", Destination: existing},
		}

		err := RenderTemplates(renderer, templates, nil, false)
		if err == nil {
			t.Error("expected error for existing file with overwrite=false, got nil")
		}

		// Original file should be untouched
		content, _ := os.ReadFile(existing)
		if string(content) != "original" {
			t.Errorf("existing file was modified despite overwrite=false")
		}
	})

	t.Run("overwrites file when overwrite is true", func(t *testing.T) {
		dir := t.TempDir()
		existing := filepath.Join(dir, "init.pp")
		if err := os.WriteFile(existing, []byte("original"), 0644); err != nil {
			t.Fatal(err)
		}

		renderer := &fakeRenderer{output: "new content"}
		templates := []TemplateFile{
			{FileName: "init.pp", Destination: existing},
		}

		if err := RenderTemplates(renderer, templates, nil, true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, _ := os.ReadFile(existing)
		if string(content) != "new content" {
			t.Errorf("got %q, want %q", string(content), "new content")
		}
	})

	t.Run("returns error when renderer fails", func(t *testing.T) {
		dir := t.TempDir()
		renderer := &fakeRenderer{err: errors.New("render failed")}
		templates := []TemplateFile{
			{FileName: "foo.pp", Destination: filepath.Join(dir, "foo.pp")},
		}

		if err := RenderTemplates(renderer, templates, nil, false); err == nil {
			t.Error("expected error from renderer, got nil")
		}
	})
}

// --- ConstructDestinationFilename ---

func TestConstructDestinationFilename_Adversarial(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty name", "", true},
		{"path traversal with ..", "..", true},
		{"dot component", ".", true},
		{"slash in name", "foo/bar", true},
		{"backslash in name", `foo\bar`, true},
		{"leading double colon", "::foo", true},
		{"trailing double colon", "foo::", true},
		{"consecutive double colons", "foo::::bar", true},
		{"valid simple name", "myclass", false},
		{"valid namespaced name", "myclass::sub", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ConstructDestinationFilename(tc.input, "mymodule", "manifests", ".pp")
			if tc.wantErr && err == nil {
				t.Errorf("expected error for input %q, got nil", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for input %q: %v", tc.input, err)
			}
		})
	}
}

// --- validateComponentName ---

func TestValidateComponentName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "mymodule", false},
		{"valid with numbers", "mymodule2", false},
		{"valid with underscores", "my_module", false},
		{"empty", "", true},
		{"slash", "foo/bar", true},
		{"backslash", `foo\bar`, true},
		{"dot dot", "..", true},
		{"single dot", ".", true},
		{"path traversal embedded", "foo/../bar", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateComponentName(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for %q, got nil", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tc.input, err)
			}
		})
	}
}
