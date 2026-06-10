// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// fakeRenderer satisfies the Renderer interface and returns configurable output.
type fakeRenderer struct {
	output string
	err    error
}

func (f *fakeRenderer) Render(_ string, _ any) (string, error) {
	return f.output, f.err
}

// makeModuleDir creates a temp directory with a minimal valid metadata.json,
// suitable for use as a WorkDir in tests.
func makeModuleDir(t *testing.T, forgeUser, moduleName string) string {
	t.Helper()
	dir := t.TempDir()
	meta := map[string]any{
		"name":                    forgeUser + "-" + moduleName,
		"version":                 "0.1.0",
		"author":                  forgeUser,
		"license":                 "Apache-2.0",
		"summary":                 "test",
		"source":                  "https://example.com",
		"dependencies":            []any{},
		"requirements":            []any{},
		"operatingsystem_support": []any{},
		"tags":                    []any{},
		"pdk-version":             "3.4.0",
	}
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// makeFunctionTemplateDir creates a temp directory with minimal function
// templates. This is necessary because NewFunction calls newRenderer
// internally and cannot accept an injected renderer.
func makeFunctionTemplateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	funcDir := filepath.Join(dir, "function")
	if err := os.MkdirAll(funcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(funcDir, "function.pp"), []byte("# {{.Name}}\nfunction {{.Name}}() >> Any {\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(funcDir, "function_spec.rb"), []byte("# frozen_string_literal: true\nrequire 'spec_helper'\ndescribe '{{.Name}}' do\nend\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// makeTaskTemplateDir creates a temp directory with minimal task templates,
// necessary because NewTask calls newRenderer internally and cannot accept an
// injected renderer.
func makeTaskTemplateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	taskDir := filepath.Join(dir, "task")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "task.sh"), []byte("#!/bin/bash\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "metadata.json"), []byte(`{"description":"","parameters":{}}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// contains reports whether substr is present in s.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
