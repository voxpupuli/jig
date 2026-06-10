// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewTest_ManifestNotFound verifies that NewTest returns an error when no
// manifest exists for the given name, and does not create any files.
func TestNewTest_ManifestNotFound(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")

	err := NewTest(ComponentOptions{
		Name:    "nosuchclass",
		WorkDir: dir,
	})
	if err == nil {
		t.Fatal("expected error for missing manifest, got nil")
	}

	specPath := filepath.Join(dir, "spec", "classes", "nosuchclass_spec.rb")
	if _, statErr := os.Stat(specPath); statErr == nil {
		t.Error("spec file should not have been created when manifest is missing")
	}
}

// TestNewTest_ManifestExistsButNoMatchingType verifies that NewTest returns an
// error when the manifest exists but contains neither a class nor a defined
// type with the expected fully-qualified name.
func TestNewTest_ManifestExistsButNoMatchingType(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")

	manifestPath := filepath.Join(dir, "manifests", "myclass.pp")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte("# empty or unrelated content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := NewTest(ComponentOptions{
		Name:    "myclass",
		WorkDir: dir,
	})
	if err == nil {
		t.Fatal("expected error when manifest contains no matching class or define, got nil")
	}
}

// TestNewTest_ClassNameSubstringFalsePositive verifies that a manifest
// containing "class mymodule::myclassextra" does not satisfy a search for
// "mymodule::myclass". This guards against the strings.Contains prefix
// collision where one name is a prefix of another.
func TestNewTest_ClassNameSubstringFalsePositive(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")

	manifestPath := filepath.Join(dir, "manifests", "myclass.pp")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		t.Fatal(err)
	}
	content := "class mymodule::myclassextra {\n}\n"
	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := NewTest(ComponentOptions{
		Name:    "myclass",
		WorkDir: dir,
	})
	if err == nil {
		t.Fatal("expected error: manifest contains a superstring of the target name, not the target itself")
	}
}

// TestNewTest_DefineNameSubstringFalsePositive is the same check for defined
// types.
func TestNewTest_DefineNameSubstringFalsePositive(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")

	manifestPath := filepath.Join(dir, "manifests", "mytype.pp")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		t.Fatal(err)
	}
	content := "define mymodule::mytypeextra {\n}\n"
	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := NewTest(ComponentOptions{
		Name:    "mytype",
		WorkDir: dir,
	})
	if err == nil {
		t.Fatal("expected error: manifest contains a superstring of the target name, not the target itself")
	}
}

// TestNewTest_SpecAlreadyExists verifies that NewTest returns an error when the
// spec file already exists, and does not overwrite it.
func TestNewTest_SpecAlreadyExists(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")

	manifestPath := filepath.Join(dir, "manifests", "myclass.pp")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte("class mymodule::myclass {\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	specPath := filepath.Join(dir, "spec", "classes", "myclass_spec.rb")
	if err := os.MkdirAll(filepath.Dir(specPath), 0755); err != nil {
		t.Fatal(err)
	}
	sentinel := []byte("# existing spec\n")
	if err := os.WriteFile(specPath, sentinel, 0644); err != nil {
		t.Fatal(err)
	}

	err := NewTest(ComponentOptions{
		Name:    "myclass",
		WorkDir: dir,
	})
	if err == nil {
		t.Fatal("expected error when spec file already exists, got nil")
	}

	// Verify the existing spec was not overwritten.
	got, readErr := os.ReadFile(specPath)
	if readErr != nil {
		t.Fatalf("could not read spec file: %v", readErr)
	}
	if string(got) != string(sentinel) {
		t.Error("existing spec file was overwritten")
	}
}

// TestNewTest_EmptyName verifies that an empty name is rejected before any
// filesystem access occurs.
func TestNewTest_EmptyName(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")

	err := NewTest(ComponentOptions{
		Name:    "",
		WorkDir: dir,
	})
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

// TestNewTest_PathTraversalInName verifies that a name containing path
// traversal sequences is rejected.
func TestNewTest_PathTraversalInName(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")

	cases := []string{"../evil", "foo/../bar", ".."}
	for _, name := range cases {
		err := NewTest(ComponentOptions{
			Name:    name,
			WorkDir: dir,
		})
		if err == nil {
			t.Errorf("expected error for path traversal name %q, got nil", name)
		}
	}
}

// TestNewTest_InvalidModuleDirectory verifies that NewTest fails cleanly when
// run outside a valid module directory.
func TestNewTest_InvalidModuleDirectory(t *testing.T) {
	err := NewTest(ComponentOptions{
		Name:    "myclass",
		WorkDir: t.TempDir(), // no metadata.json
	})
	if err == nil {
		t.Fatal("expected error for invalid module directory, got nil")
	}
}

// TestNewTest_WrongModuleNameInManifest verifies that a manifest whose class
// declaration uses a different module name is not accepted. This guards against
// a scenario where a manifest was copied from another module and the name was
// not updated.
func TestNewTest_WrongModuleNameInManifest(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")

	manifestPath := filepath.Join(dir, "manifests", "myclass.pp")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		t.Fatal(err)
	}
	content := "class othermodule::myclass {\n}\n"
	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := NewTest(ComponentOptions{
		Name:    "myclass",
		WorkDir: dir,
	})
	if err == nil {
		t.Fatal("expected error when manifest uses a different module name, got nil")
	}
}

// TestNewTest_InitClass verifies that "init" is correctly resolved to the
// module's root class, whose declaration is "class mymodule {" rather than
// "class mymodule::init {".
func TestNewTest_InitClass(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")

	manifestPath := filepath.Join(dir, "manifests", "init.pp")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte("class mymodule {\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := NewTest(ComponentOptions{
		Name:    "init",
		WorkDir: dir,
	})
	if err != nil {
		t.Fatalf("unexpected error for init class: %v", err)
	}

	specPath := filepath.Join(dir, "spec", "classes", "init_spec.rb")
	if _, err := os.Stat(specPath); err != nil {
		t.Errorf("expected spec file at %s: %v", specPath, err)
	}
}
