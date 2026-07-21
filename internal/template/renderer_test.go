// SPDX-License-Identifier: GPL-3.0-or-later
package template

import (
	"os"
	"path/filepath"
	"slices"
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
	t.Run("renders empty verbatim file", func(t *testing.T) {
		r := NewRenderer()
		out, err := r.Render("module/data/.gitkeep", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != "" {
			t.Errorf("expected empty output, got %q", out)
		}
	})

	t.Run("logical name resolves .tmpl variant and renders with data", func(t *testing.T) {
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

	t.Run("verbatim file is returned as-is", func(t *testing.T) {
		r := NewRenderer()
		out, err := r.Render("module/Gemfile", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		embedded, err := embeddedTemplates.ReadFile("templates/module/Gemfile")
		if err != nil {
			t.Fatal(err)
		}
		if out != string(embedded) {
			t.Errorf("verbatim output does not match embedded content")
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
	t.Run("external .tmpl overrides embedded and is rendered", func(t *testing.T) {
		dir := t.TempDir()
		writeExternalTemplate(t, dir, "module/hiera.yaml.tmpl", "custom {{.ModuleName}}")

		r := NewRendererWithExternalDir(dir)
		out, err := r.Render("module/hiera.yaml", struct{ ModuleName string }{"mymodule"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != "custom mymodule" {
			t.Errorf("expected %q, got %q", "custom mymodule", out)
		}
	})

	t.Run("external verbatim file overrides embedded .tmpl by destination", func(t *testing.T) {
		dir := t.TempDir()
		// Embedded has module/README.md.tmpl; a plain external README.md
		// must win and be copied without rendering.
		writeExternalTemplate(t, dir, "module/README.md", "plain readme")

		r := NewRendererWithExternalDir(dir)
		out, err := r.Render("module/README.md", struct{ ModuleName string }{"mymodule"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != "plain readme" {
			t.Errorf("expected %q, got %q", "plain readme", out)
		}
	})

	t.Run("both variants in external dir is an error", func(t *testing.T) {
		dir := t.TempDir()
		writeExternalTemplate(t, dir, "module/hiera.yaml", "plain")
		writeExternalTemplate(t, dir, "module/hiera.yaml.tmpl", "templated")

		r := NewRendererWithExternalDir(dir)
		_, err := r.Render("module/hiera.yaml", nil)
		if err == nil || !strings.Contains(err.Error(), "would produce the same file") {
			t.Errorf("expected same-destination collision error, got %v", err)
		}
	})

	t.Run("falls back to embedded when external file missing", func(t *testing.T) {
		dir := t.TempDir()
		// External dir exists but has no templates in it

		r := NewRendererWithExternalDir(dir)
		out, err := r.Render("module/data/.gitkeep", nil)
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
		out, err := r.Render("module/data/.gitkeep", nil)
		if err != nil {
			t.Fatalf("unexpected error on fallback from nonexistent dir: %v", err)
		}
		if out != "" {
			t.Errorf("expected empty output from embedded fallback, got %q", out)
		}
	})

	t.Run("external file unreadable returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "module", "hiera.yaml")
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
		_, err := r.Render("module/hiera.yaml", nil)
		if err == nil {
			t.Error("expected error for unreadable external template, got nil")
		}
	})

	t.Run("external template with invalid syntax returns error", func(t *testing.T) {
		dir := t.TempDir()
		writeExternalTemplate(t, dir, "module/hiera.yaml.tmpl", "{{ .Unclosed")

		r := NewRendererWithExternalDir(dir)
		_, err := r.Render("module/hiera.yaml", nil)
		if err == nil {
			t.Error("expected error for invalid template syntax, got nil")
		}
	})

	t.Run("verbatim file with template delimiters is copied unrendered", func(t *testing.T) {
		// A pre-2.0 style tree: template vars in a file without the .tmpl
		// suffix. It must be copied as-is.
		dir := t.TempDir()
		writeExternalTemplate(t, dir, "module/hiera.yaml", "hello {{.ModuleName}}")

		r := NewRendererWithExternalDir(dir)
		out, err := r.Render("module/hiera.yaml", struct{ ModuleName string }{"mymodule"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != "hello {{.ModuleName}}" {
			t.Errorf("expected verbatim copy, got %q", out)
		}
	})
}

// --- Template data handling ---

func TestRender_MissingDataField(t *testing.T) {
	// Go's text/template silently produces <no value> for missing fields
	// rather than erroring. This documents that behavior -- if missingkey=error
	// is ever added, these tests will need updating.
	dir := t.TempDir()
	writeExternalTemplate(t, dir, "module/hiera.yaml.tmpl", "hello {{.MissingField}}")

	r := NewRendererWithExternalDir(dir)
	out, err := r.Render("module/hiera.yaml", nil)
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
		{"dot dot in component", "module/../../etc/passwd"},
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

// --- ListTree ---

func TestListTree(t *testing.T) {
	t.Run("embedded module tree uses logical destination paths", func(t *testing.T) {
		r := NewRenderer()
		names, err := r.ListTree("module")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{
			".devcontainer/devcontainer.json",
			".gitignore",
			"README.md",
			"data/.gitkeep",
			"manifests/init.pp",
			"spec/classes/init_spec.rb",
		}
		for _, want := range expected {
			if !slices.Contains(names, want) {
				t.Errorf("expected ListTree to contain %q, got %v", want, names)
			}
		}
		for _, name := range names {
			if strings.HasSuffix(name, TmplSuffix) {
				t.Errorf("logical name %q must not keep the %s suffix", name, TmplSuffix)
			}
		}
	})

	t.Run("external files are unioned in", func(t *testing.T) {
		dir := t.TempDir()
		writeExternalTemplate(t, dir, "module/.github/workflows/ci.yml", "on: push\n")
		writeExternalTemplate(t, dir, "module/CONTRIBUTING.md.tmpl", "# {{.ModuleName}}\n")

		r := NewRendererWithExternalDir(dir)
		names, err := r.ListTree("module")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, want := range []string{".github/workflows/ci.yml", "CONTRIBUTING.md", ".gitignore"} {
			if !slices.Contains(names, want) {
				t.Errorf("expected ListTree to contain %q, got %v", want, names)
			}
		}
	})

	t.Run("external override does not duplicate the entry", func(t *testing.T) {
		dir := t.TempDir()
		writeExternalTemplate(t, dir, "module/README.md", "plain readme")

		r := NewRendererWithExternalDir(dir)
		names, err := r.ListTree("module")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := 0
		for _, name := range names {
			if name == "README.md" {
				count++
			}
		}
		if count != 1 {
			t.Errorf("expected exactly one README.md entry, got %d in %v", count, names)
		}
	})

	t.Run("both variants in one source is an error", func(t *testing.T) {
		dir := t.TempDir()
		writeExternalTemplate(t, dir, "module/foo.yml", "plain")
		writeExternalTemplate(t, dir, "module/foo.yml.tmpl", "templated")

		r := NewRendererWithExternalDir(dir)
		_, err := r.ListTree("module")
		if err == nil || !strings.Contains(err.Error(), "would produce the same file") {
			t.Errorf("expected same-destination collision error, got %v", err)
		}
	})

	t.Run("results are sorted", func(t *testing.T) {
		r := NewRenderer()
		names, err := r.ListTree("module")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !slices.IsSorted(names) {
			t.Errorf("expected sorted names, got %v", names)
		}
	})

	t.Run("invalid root returns error", func(t *testing.T) {
		r := NewRenderer()
		if _, err := r.ListTree("../etc"); err == nil {
			t.Error("expected error for traversal root, got nil")
		}
	})
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
			"module/manifests/init.pp.tmpl",
			"module/README.md.tmpl",
			"module/Gemfile",
			"module/Rakefile.tmpl",
			"module/hiera.yaml",
			"module/.gitignore",
			"module/.devcontainer/devcontainer.json",
			"module/data/.gitkeep",
			"class/class.pp.tmpl",
			"class/class_spec.rb.tmpl",
			"type/defined_type.pp.tmpl",
			"type/defined_type_spec.rb.tmpl",
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
		dumped, err := os.ReadFile(filepath.Join(dest, "module/Gemfile"))
		if err != nil {
			t.Fatalf("could not read dumped file: %v", err)
		}

		embedded, err := embeddedTemplates.ReadFile("templates/module/Gemfile")
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
