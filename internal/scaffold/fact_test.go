// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewFact_HappyPath verifies that a valid name produces both expected files
// at the correct locations.
func TestNewFact_HappyPath(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	err := NewFact(ComponentOptions{
		Name:        "myfact",
		WorkDir:     dir,
		TemplateDir: makeFactTemplateDir(t),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFiles := []string{
		filepath.Join(dir, "lib", "facter", "myfact.rb"),
		filepath.Join(dir, "spec", "unit", "facter", "myfact_spec.rb"),
	}
	for _, f := range expectedFiles {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("expected file %s to exist: %v", f, err)
		}
	}
}

// TestNewFact_RejectsInvalidModuleDirectory verifies that NewFact fails before
// creating any files when run outside a valid module directory.
func TestNewFact_RejectsInvalidModuleDirectory(t *testing.T) {
	emptyDir := t.TempDir()
	err := NewFact(ComponentOptions{
		Name:        "myfact",
		WorkDir:     emptyDir,
		TemplateDir: makeFactTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for missing metadata.json, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(emptyDir, "lib")); statErr == nil {
		t.Error("lib directory should not have been created in an invalid module directory")
	}
}

// TestNewFact_RejectsCorruptMetadata verifies that malformed JSON in
// metadata.json is caught rather than silently ignored.
func TestNewFact_RejectsCorruptMetadata(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte("{{{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	err := NewFact(ComponentOptions{
		Name:        "myfact",
		WorkDir:     dir,
		TemplateDir: makeFactTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for corrupt metadata.json, got nil")
	}
}

// TestNewFact_RejectsNamespacedName verifies that a name containing "::" is
// rejected, since namespacing is not valid for facts.
func TestNewFact_RejectsNamespacedName(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	err := NewFact(ComponentOptions{
		Name:        "my::fact",
		WorkDir:     dir,
		TemplateDir: makeFactTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for namespaced fact name, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "lib", "facter", "my::fact.rb")); statErr == nil {
		t.Error("fact file should not have been created for namespaced name")
	}
}

// TestNewFact_RejectsNameWithPathSeparator verifies that a name containing a
// slash or backslash cannot be used to write files outside the facter directory.
func TestNewFact_RejectsNameWithPathSeparator(t *testing.T) {
	cases := []string{"foo/bar", `foo\bar`}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			dir := makeModuleDir(t, "myuser", "mymodule")
			err := NewFact(ComponentOptions{
				Name:        name,
				WorkDir:     dir,
				TemplateDir: makeFactTemplateDir(t),
			})
			if err == nil {
				t.Errorf("expected error for fact name %q with path separator, got nil", name)
			}
		})
	}
}

// TestNewFact_RejectsPathTraversal verifies that a name like "../evil" does not
// write files outside the expected lib/facter/ directory. filepath.Join will
// clean the path silently, so without explicit validation the file lands at
// lib/evil.rb rather than erroring -- this test documents that gap.
func TestNewFact_RejectsPathTraversal(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	err := NewFact(ComponentOptions{
		Name:        "../evil",
		WorkDir:     dir,
		TemplateDir: makeFactTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for path traversal in fact name, got nil")
	}
	// Confirm nothing was written outside lib/facter/
	if _, statErr := os.Stat(filepath.Join(dir, "lib", "evil.rb")); statErr == nil {
		t.Error("file was written outside lib/facter/ via path traversal")
	}
}

// TestNewFact_RejectsEmptyName documents the current behavior for an empty
// name. Without explicit validation, NewFact will attempt to create ".rb",
// which is not a useful file. This test exists to pin the behavior and prompt
// adding an early empty-name check if that is the desired fix.
func TestNewFact_RejectsEmptyName(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	err := NewFact(ComponentOptions{
		Name:        "",
		WorkDir:     dir,
		TemplateDir: makeFactTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for empty fact name, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "lib", "facter", ".rb")); statErr == nil {
		t.Error("a file named '.rb' was created from an empty fact name")
	}
}

// TestNewFact_RefusesIfFactFileExists_NoSpecCreated verifies that a
// pre-existing fact file causes an early error and the spec file is not created
// as a side effect, leaving no partial state on disk.
func TestNewFact_RefusesIfFactFileExists_NoSpecCreated(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	factPath := filepath.Join(dir, "lib", "facter", "myfact.rb")
	if err := os.MkdirAll(filepath.Dir(factPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(factPath, []byte("# existing"), 0644); err != nil {
		t.Fatal(err)
	}

	err := NewFact(ComponentOptions{
		Name:        "myfact",
		WorkDir:     dir,
		TemplateDir: makeFactTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for existing fact file, got nil")
	}

	specPath := filepath.Join(dir, "spec", "unit", "facter", "myfact_spec.rb")
	if _, statErr := os.Stat(specPath); statErr == nil {
		t.Error("spec file should not have been created when fact file already exists")
	}
}

// TestNewFact_RefusesIfSpecFileExists_NoFactCreated verifies the inverse: a
// pre-existing spec file causes an error and the fact file is not written.
// This guards against partial state when only the spec was left behind.
func TestNewFact_RefusesIfSpecFileExists_NoFactCreated(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	specPath := filepath.Join(dir, "spec", "unit", "facter", "myfact_spec.rb")
	if err := os.MkdirAll(filepath.Dir(specPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(specPath, []byte("# existing"), 0644); err != nil {
		t.Fatal(err)
	}

	err := NewFact(ComponentOptions{
		Name:        "myfact",
		WorkDir:     dir,
		TemplateDir: makeFactTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for existing spec file, got nil")
	}

	factPath := filepath.Join(dir, "lib", "facter", "myfact.rb")
	if _, statErr := os.Stat(factPath); statErr == nil {
		t.Error("fact file should not have been created when spec file already exists")
	}
}

// TestNewFact_ExistingFactFileIsUntouched verifies that the content of a
// pre-existing fact file is not modified when NewFact returns an error.
func TestNewFact_ExistingFactFileIsUntouched(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	factPath := filepath.Join(dir, "lib", "facter", "myfact.rb")
	if err := os.MkdirAll(filepath.Dir(factPath), 0755); err != nil {
		t.Fatal(err)
	}
	original := []byte("# do not touch")
	if err := os.WriteFile(factPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	_ = NewFact(ComponentOptions{
		Name:        "myfact",
		WorkDir:     dir,
		TemplateDir: makeFactTemplateDir(t),
	})

	content, err := os.ReadFile(factPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(original) {
		t.Errorf("existing fact file was modified: got %q, want %q", string(content), string(original))
	}
}
