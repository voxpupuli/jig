// SPDX-License-Identifier: GPL-3.0-or-later
package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeExternalTemplate creates a template file in dir at the given relative
// path with the given content. Helper for external dir tests.
func writeExternalTemplate(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// --- NewRenderer / NewRendererWithExternalDir ---

func TestNewRenderer(t *testing.T) {
	r := NewRenderer()
	if r == nil {
		t.Fatal("expected non-nil Renderer")
	}
	if r.externalDir != "" {
		t.Errorf("expected empty externalDir, got %q", r.externalDir)
	}
}

func TestNewRendererWithExternalDir(t *testing.T) {
	r := NewRendererWithExternalDir("/some/path")
	if r == nil {
		t.Fatal("expected non-nil Renderer")
	}
	if r.externalDir != "/some/path" {
		t.Errorf("expected externalDir %q, got %q", "/some/path", r.externalDir)
	}
}

// --- Render with embedded templates ---

func TestRender_EmbeddedTemplate(t *testing.T) {
	t.Run("renders empty template", func(t *testing.T) {
		r := NewRenderer()
		out, err := r.Render("common/gitkeep", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != "" {
			t.Errorf("expected empty output, got %q", out)
		}
	})

	t.Run("renders template with data", func(t *testing.T) {
		r := NewRenderer()
		data := struct{ ModuleName string }{ModuleName: "mymodule"}
		out, err := r.Render("module/manifests/init.pp", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out, "mymodule") {
			t.Errorf("expected output to contain %q, got %q", "mymodule", out)
		}
	})

	t.Run("missing embedded template returns error", func(t *testing.T) {
		r := NewRenderer()
		_, err := r.Render("nonexistent/template.pp", nil)
		if err == nil {
			t.Error("expected error for missing embedded template, got nil")
		}
	})

	// Regression: on Windows, filepath.Clean rewrites "/" to "\", producing a
	// backslash-separated name that embed.FS (which always uses "/") cannot
	// find -- e.g. "templates/module\manifests\init.pp". Feeding a
	// backslash-separated name here reproduces that state on any OS and must
	// still resolve. See https://github.com/voxpupuli/jig -- Windows path bug.
	t.Run("backslash-separated name resolves embedded template", func(t *testing.T) {
		r := NewRenderer()
		data := struct{ ModuleName string }{ModuleName: "mymodule"}
		out, err := r.Render(`module\manifests\init.pp`, data)
		if err != nil {
			t.Fatalf("unexpected error for backslash-separated name: %v", err)
		}
		if !strings.Contains(out, "mymodule") {
			t.Errorf("expected output to contain %q, got %q", "mymodule", out)
		}
	})
}

// --- Render with external dir ---

func TestRender_ExternalDir(t *testing.T) {
	t.Run("uses external template when present", func(t *testing.T) {
		dir := t.TempDir()
		writeExternalTemplate(t, dir, "common/gitkeep", "custom content")

		r := NewRendererWithExternalDir(dir)
		out, err := r.Render("common/gitkeep", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != "custom content" {
			t.Errorf("expected %q, got %q", "custom content", out)
		}
	})

	t.Run("falls back to embedded when external file missing", func(t *testing.T) {
		dir := t.TempDir()
		// External dir exists but has no templates in it

		r := NewRendererWithExternalDir(dir)
		out, err := r.Render("common/gitkeep", nil)
		if err != nil {
			t.Fatalf("unexpected error on fallback: %v", err)
		}
		// Embedded gitkeep is empty
		if out != "" {
			t.Errorf("expected empty output from embedded fallback, got %q", out)
		}
	})

	t.Run("external dir does not exist falls back to embedded", func(t *testing.T) {
		r := NewRendererWithExternalDir("/nonexistent/dir")
		// The external file won't be found (IsNotExist), should fall back
		out, err := r.Render("common/gitkeep", nil)
		if err != nil {
			t.Fatalf("unexpected error on fallback from nonexistent dir: %v", err)
		}
		if out != "" {
			t.Errorf("expected empty output from embedded fallback, got %q", out)
		}
	})

	t.Run("external file unreadable returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "common", "gitkeep")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
		// Make it unreadable
		if err := os.Chmod(path, 0000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(path, 0644) })

		r := NewRendererWithExternalDir(dir)
		_, err := r.Render("common/gitkeep", nil)
		if err == nil {
			t.Error("expected error for unreadable external template, got nil")
		}
	})

	t.Run("external template with invalid syntax returns error", func(t *testing.T) {
		dir := t.TempDir()
		writeExternalTemplate(t, dir, "common/gitkeep", "{{ .Unclosed")

		r := NewRendererWithExternalDir(dir)
		_, err := r.Render("common/gitkeep", nil)
		if err == nil {
			t.Error("expected error for invalid template syntax, got nil")
		}
	})
}

// --- Template data handling ---

func TestRender_MissingDataField(t *testing.T) {
	// Go's text/template silently produces <no value> for missing fields
	// rather than erroring. This documents that behavior -- if missingkey=error
	// is ever added, these tests will need updating.
	dir := t.TempDir()
	writeExternalTemplate(t, dir, "common/gitkeep", "hello {{.MissingField}}")

	r := NewRendererWithExternalDir(dir)
	out, err := r.Render("common/gitkeep", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "<no value>") {
		t.Errorf("expected '<no value>' for missing field, got %q", out)
	}
}

// --- Adversarial cases ---

func TestRender_PathTraversal(t *testing.T) {
	// filepath.Join(externalDir, templateName) with a traversal sequence in
	// templateName can escape the external directory. The embedded FS path is
	// safe because embed.FS rejects ".." components internally, but the
	// external dir read has no such protection.
	cases := []struct {
		name         string
		templateName string
	}{
		{"dot dot", "../../etc/passwd"},
		{"dot dot in component", "common/../../etc/passwd"},
		{"absolute path", "/etc/passwd"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			r := NewRendererWithExternalDir(dir)
			// Should either error or fall back safely to embedded -- it must
			// not successfully read a file outside the external dir.
			_, err := r.Render(tc.templateName, nil)
			// We expect an error because the traversal target won't be a
			// valid template in the embedded FS either. The important thing
			// is that it doesn't silently read an arbitrary file.
			if err == nil {
				t.Errorf("expected error for traversal path %q, got nil -- potential path traversal vulnerability", tc.templateName)
			}
		})
	}
}

func TestRender_EmptyTemplateName(t *testing.T) {
	r := NewRenderer()
	_, err := r.Render("", nil)
	if err == nil {
		t.Error("expected error for empty template name, got nil")
	}
}

// --- DumpTemplates ---

func TestDumpTemplates(t *testing.T) {
	t.Run("writes all embedded templates to disk", func(t *testing.T) {
		dest := t.TempDir()
		if err := DumpTemplates(dest); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Spot-check a known set of files
		expectedFiles := []string{
			"common/gitkeep",
			"module/manifests/init.pp",
			"module/README.md",
			"module/Gemfile",
			"module/Rakefile",
			"module/hiera.yaml",
			"class/class.pp",
			"class/class_spec.rb",
			"type/defined_type.pp",
			"type/defined_type_spec.rb",
		}
		for _, f := range expectedFiles {
			path := filepath.Join(dest, f)
			if _, err := os.Stat(path); err != nil {
				t.Errorf("expected dumped file %s to exist: %v", f, err)
			}
		}
	})

	t.Run("dumped templates match embedded content", func(t *testing.T) {
		dest := t.TempDir()
		if err := DumpTemplates(dest); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Read a dumped file and compare to what the renderer produces
		dumped, err := os.ReadFile(filepath.Join(dest, "common/gitkeep"))
		if err != nil {
			t.Fatalf("could not read dumped file: %v", err)
		}

		embedded, err := embeddedTemplates.ReadFile("templates/common/gitkeep")
		if err != nil {
			t.Fatalf("could not read embedded file: %v", err)
		}

		if string(dumped) != string(embedded) {
			t.Errorf("dumped content does not match embedded content")
		}
	})

	t.Run("dump to unwritable destination returns error", func(t *testing.T) {
		dest := t.TempDir()
		if err := os.Chmod(dest, 0555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(dest, 0755) })

		err := DumpTemplates(dest)
		if err == nil {
			t.Error("expected error writing to unwritable destination, got nil")
		}
	})
}
