// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewFunction_CreatesFilesAtExpectedLocations(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	err := NewFunction(ComponentOptions{
		Name:        "myfunc",
		WorkDir:     dir,
		TemplateDir: makeFunctionTemplateDir(t),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFiles := []string{
		filepath.Join(dir, "functions", "myfunc.pp"),
		filepath.Join(dir, "spec", "functions", "myfunc_spec.rb"),
	}
	for _, f := range expectedFiles {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("expected file %s to exist: %v", f, err)
		}
	}
}

// TestNewFunction_OutputContainsFullyQualifiedName verifies that the rendered
// files reference "module::function", not just the bare name passed in opts.Name.
// This tests the contract between NewFunction and the templates: if functionName
// were accidentally replaced with opts.Name in the data struct, this catches it.
func TestNewFunction_OutputContainsFullyQualifiedName(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	err := NewFunction(ComponentOptions{
		Name:        "myfunc",
		WorkDir:     dir,
		TemplateDir: makeFunctionTemplateDir(t),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	funcContent, err := os.ReadFile(filepath.Join(dir, "functions", "myfunc.pp"))
	if err != nil {
		t.Fatalf("could not read function file: %v", err)
	}
	if !contains(string(funcContent), "mymodule::myfunc") {
		t.Errorf("function file content %q does not contain fully-qualified name %q", string(funcContent), "mymodule::myfunc")
	}

	specContent, err := os.ReadFile(filepath.Join(dir, "spec", "functions", "myfunc_spec.rb"))
	if err != nil {
		t.Fatalf("could not read spec file: %v", err)
	}
	if !contains(string(specContent), "mymodule::myfunc") {
		t.Errorf("spec file content %q does not contain fully-qualified name %q", string(specContent), "mymodule::myfunc")
	}
}

// TestNewFunction_RejectsInvalidModuleDirectory verifies that NewFunction
// fails before creating any files when run outside a valid module directory.
func TestNewFunction_RejectsInvalidModuleDirectory(t *testing.T) {
	emptyDir := t.TempDir()
	err := NewFunction(ComponentOptions{
		Name:        "myfunc",
		WorkDir:     emptyDir,
		TemplateDir: makeFunctionTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for missing metadata.json, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(emptyDir, "functions", "myfunc.pp")); statErr == nil {
		t.Error("function file should not have been created in an invalid module directory")
	}
}

// TestNewFunction_RejectsCorruptMetadata verifies that malformed JSON in
// metadata.json is caught and not silently ignored.
func TestNewFunction_RejectsCorruptMetadata(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte("{{{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	err := NewFunction(ComponentOptions{
		Name:        "myfunc",
		WorkDir:     dir,
		TemplateDir: makeFunctionTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for corrupt metadata.json, got nil")
	}
}

// TestNewFunction_RejectsEmptyName verifies that an empty name is rejected
// before any filesystem work is done.
func TestNewFunction_RejectsEmptyName(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	err := NewFunction(ComponentOptions{
		Name:        "",
		WorkDir:     dir,
		TemplateDir: makeFunctionTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for empty function name, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "functions")); statErr == nil {
		t.Error("functions directory should not have been created for empty name")
	}
}

// TestNewFunction_RejectsNameEqualToModuleName verifies that passing the
// module name as the function name is caught. ConstructDestinationFilename
// rejects it, but this test pins the behavior at the NewFunction level.
func TestNewFunction_RejectsNameEqualToModuleName(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	err := NewFunction(ComponentOptions{
		Name:        "mymodule",
		WorkDir:     dir,
		TemplateDir: makeFunctionTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error when function name equals module name, got nil")
	}
}

// TestNewFunction_RejectsNameWithModulePrefix verifies that a user passing the
// fully-qualified name (e.g. "mymodule::myfunc") instead of just the
// unqualified name ("myfunc") is rejected.
func TestNewFunction_RejectsNameWithModulePrefix(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	err := NewFunction(ComponentOptions{
		Name:        "mymodule::myfunc",
		WorkDir:     dir,
		TemplateDir: makeFunctionTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error when function name includes the module prefix, got nil")
	}
}

// TestNewFunction_RejectsPathSeparatorInName verifies that a name containing
// a slash or backslash cannot be used to write files outside the functions
// directory.
func TestNewFunction_RejectsPathSeparatorInName(t *testing.T) {
	cases := []string{"foo/bar", `foo\bar`}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			dir := makeModuleDir(t, "myuser", "mymodule")
			err := NewFunction(ComponentOptions{
				Name:        name,
				WorkDir:     dir,
				TemplateDir: makeFunctionTemplateDir(t),
			})
			if err == nil {
				t.Errorf("expected error for name %q with path separator, got nil", name)
			}
		})
	}
}

// TestNewFunction_RefusesIfFunctionFileExists_NoSpecCreated verifies that an
// existing function file causes an early error and that the spec file is not
// created as a side effect, leaving no partial state.
func TestNewFunction_RefusesIfFunctionFileExists_NoSpecCreated(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	funcPath := filepath.Join(dir, "functions", "myfunc.pp")
	if err := os.MkdirAll(filepath.Dir(funcPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(funcPath, []byte("# existing"), 0644); err != nil {
		t.Fatal(err)
	}

	err := NewFunction(ComponentOptions{
		Name:        "myfunc",
		WorkDir:     dir,
		TemplateDir: makeFunctionTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for existing function file, got nil")
	}

	specPath := filepath.Join(dir, "spec", "functions", "myfunc_spec.rb")
	if _, statErr := os.Stat(specPath); statErr == nil {
		t.Error("spec file should not have been created when function file already exists")
	}
}

// TestNewFunction_RefusesIfSpecFileExists_NoFunctionFileCreated verifies the
// inverse: an existing spec file causes an error and the function file is not
// written. This guards against partial state when only the spec was left behind
// by a previous failed run.
func TestNewFunction_RefusesIfSpecFileExists_NoFunctionFileCreated(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	specPath := filepath.Join(dir, "spec", "functions", "myfunc_spec.rb")
	if err := os.MkdirAll(filepath.Dir(specPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(specPath, []byte("# existing"), 0644); err != nil {
		t.Fatal(err)
	}

	err := NewFunction(ComponentOptions{
		Name:        "myfunc",
		WorkDir:     dir,
		TemplateDir: makeFunctionTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for existing spec file, got nil")
	}

	funcPath := filepath.Join(dir, "functions", "myfunc.pp")
	if _, statErr := os.Stat(funcPath); statErr == nil {
		t.Error("function file should not have been created when spec file already exists")
	}
}

// TestNewFunction_NamespacedCreatesCorrectDirectoryStructure verifies that a
// namespaced name like "sub::myfunc" produces the correct nested paths under
// functions/ and spec/functions/, mirroring the behavior of NewClass.
func TestNewFunction_NamespacedCreatesCorrectDirectoryStructure(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	err := NewFunction(ComponentOptions{
		Name:        "sub::myfunc",
		WorkDir:     dir,
		TemplateDir: makeFunctionTemplateDir(t),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFiles := []string{
		filepath.Join(dir, "functions", "sub", "myfunc.pp"),
		filepath.Join(dir, "spec", "functions", "sub", "myfunc_spec.rb"),
	}
	for _, f := range expectedFiles {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("expected file %s to exist: %v", f, err)
		}
	}
}

// TestNewFunction_NamespacedOutputContainsFullyQualifiedName verifies that a
// namespaced function's output uses the full name "module::sub::function", not
// "module::function" with the intermediate namespace silently dropped.
func TestNewFunction_NamespacedOutputContainsFullyQualifiedName(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	err := NewFunction(ComponentOptions{
		Name:        "sub::myfunc",
		WorkDir:     dir,
		TemplateDir: makeFunctionTemplateDir(t),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "functions", "sub", "myfunc.pp"))
	if err != nil {
		t.Fatalf("could not read function file: %v", err)
	}
	if !contains(string(content), "mymodule::sub::myfunc") {
		t.Errorf("function file content %q does not contain fully-qualified name %q", string(content), "mymodule::sub::myfunc")
	}
}

// makeFactTemplateDir creates a temp directory with minimal fact templates.
func makeFactTemplateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	factDir := filepath.Join(dir, "fact")
	if err := os.MkdirAll(factDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(factDir, "fact.rb"), []byte("# {{.Name}}\nFacter.add('{{.Name}}') do\n  setcode { nil }\nend\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(factDir, "fact_spec.rb"), []byte("# frozen_string_literal: true\nrequire 'spec_helper'\ndescribe '{{.Name}}' do\nend\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}
