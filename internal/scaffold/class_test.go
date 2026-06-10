// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewClass(t *testing.T) {
	t.Run("creates class and spec files", func(t *testing.T) {
		dir := makeModuleDir(t, "myuser", "mymodule")

		opts := ComponentOptions{
			Name:    "myclass",
			WorkDir: dir,
		}

		if err := NewClass(opts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedFiles := []string{
			filepath.Join(dir, "manifests", "myclass.pp"),
			filepath.Join(dir, "spec", "classes", "myclass_spec.rb"),
		}
		for _, f := range expectedFiles {
			if _, err := os.Stat(f); err != nil {
				t.Errorf("expected file %s to exist: %v", f, err)
			}
		}
	})

	t.Run("creates namespaced class and spec files", func(t *testing.T) {
		dir := makeModuleDir(t, "myuser", "mymodule")

		opts := ComponentOptions{
			Name:    "myclass::sub",
			WorkDir: dir,
		}

		if err := NewClass(opts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedFiles := []string{
			filepath.Join(dir, "manifests", "myclass", "sub.pp"),
			filepath.Join(dir, "spec", "classes", "myclass", "sub_spec.rb"),
		}
		for _, f := range expectedFiles {
			if _, err := os.Stat(f); err != nil {
				t.Errorf("expected file %s to exist: %v", f, err)
			}
		}
	})

	t.Run("returns error when class file already exists", func(t *testing.T) {
		dir := makeModuleDir(t, "myuser", "mymodule")

		classPath := filepath.Join(dir, "manifests", "myclass.pp")
		if err := os.MkdirAll(filepath.Dir(classPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(classPath, []byte("existing"), 0644); err != nil {
			t.Fatal(err)
		}

		opts := ComponentOptions{
			Name:    "myclass",
			WorkDir: dir,
		}

		if err := NewClass(opts); err == nil {
			t.Error("expected error for existing class file, got nil")
		}
	})

	t.Run("returns error when name includes module name", func(t *testing.T) {
		dir := makeModuleDir(t, "myuser", "mymodule")

		opts := ComponentOptions{
			Name:    "mymodule::myclass",
			WorkDir: dir,
		}

		if err := NewClass(opts); err == nil {
			t.Error("expected error when class name includes module name, got nil")
		}
	})

	t.Run("returns error for invalid module directory", func(t *testing.T) {
		opts := ComponentOptions{
			Name:    "myclass",
			WorkDir: t.TempDir(),
		}

		if err := NewClass(opts); err == nil {
			t.Error("expected error for missing metadata.json, got nil")
		}
	})
}

func TestNewClass_EmptyName(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	opts := ComponentOptions{
		Name:    "",
		WorkDir: dir,
	}
	if err := NewClass(opts); err == nil {
		t.Error("expected error for empty class name, got nil")
	}
	if _, err := os.Stat(filepath.Join(dir, "manifests", ".pp")); err == nil {
		t.Error("a file named '.pp' was created from an empty class name")
	}
}
