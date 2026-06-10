// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDefinedType(t *testing.T) {
	t.Run("creates defined type and spec files", func(t *testing.T) {
		dir := makeModuleDir(t, "myuser", "mymodule")

		opts := ComponentOptions{
			Name:    "mytype",
			WorkDir: dir,
		}

		if err := NewDefinedType(opts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedFiles := []string{
			filepath.Join(dir, "manifests", "mytype.pp"),
			filepath.Join(dir, "spec", "defines", "mytype_spec.rb"),
		}
		for _, f := range expectedFiles {
			if _, err := os.Stat(f); err != nil {
				t.Errorf("expected file %s to exist: %v", f, err)
			}
		}
	})

	t.Run("returns error when type file already exists", func(t *testing.T) {
		dir := makeModuleDir(t, "myuser", "mymodule")

		typePath := filepath.Join(dir, "manifests", "mytype.pp")
		if err := os.MkdirAll(filepath.Dir(typePath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(typePath, []byte("existing"), 0644); err != nil {
			t.Fatal(err)
		}

		opts := ComponentOptions{
			Name:    "mytype",
			WorkDir: dir,
		}

		if err := NewDefinedType(opts); err == nil {
			t.Error("expected error for existing type file, got nil")
		}
	})

	t.Run("returns error when spec file already exists", func(t *testing.T) {
		dir := makeModuleDir(t, "myuser", "mymodule")

		specPath := filepath.Join(dir, "spec", "defines", "mytype_spec.rb")
		if err := os.MkdirAll(filepath.Dir(specPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(specPath, []byte("existing"), 0644); err != nil {
			t.Fatal(err)
		}

		opts := ComponentOptions{
			Name:    "mytype",
			WorkDir: dir,
		}

		if err := NewDefinedType(opts); err == nil {
			t.Error("expected error for existing spec file, got nil")
		}
	})
}

func TestNewDefinedType_EmptyName(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	opts := ComponentOptions{
		Name:    "",
		WorkDir: dir,
	}
	if err := NewDefinedType(opts); err == nil {
		t.Error("expected error for empty defined type name, got nil")
	}
}
