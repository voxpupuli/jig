// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewTask_HappyPath verifies that a valid name produces both expected files.
func TestNewTask_HappyPath(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	err := NewTask(ComponentOptions{
		Name:        "mytask",
		WorkDir:     dir,
		TemplateDir: makeTaskTemplateDir(t),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFiles := []string{
		filepath.Join(dir, "tasks", "mytask.sh"),
		filepath.Join(dir, "tasks", "mytask.json"),
	}
	for _, f := range expectedFiles {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("expected file %s to exist: %v", f, err)
		}
	}
}

// TestNewTask_InitIsValid verifies that the special name "init" is accepted,
// since it maps to the module itself and is a valid task name.
func TestNewTask_InitIsValid(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	err := NewTask(ComponentOptions{
		Name:        "init",
		WorkDir:     dir,
		TemplateDir: makeTaskTemplateDir(t),
	})
	if err != nil {
		t.Errorf("expected init to be a valid task name, got error: %v", err)
	}
}

// TestNewTask_RejectsInvalidModuleDirectory verifies that NewTask fails before
// touching the filesystem when run outside a valid module directory.
func TestNewTask_RejectsInvalidModuleDirectory(t *testing.T) {
	emptyDir := t.TempDir()
	err := NewTask(ComponentOptions{
		Name:        "mytask",
		WorkDir:     emptyDir,
		TemplateDir: makeTaskTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for missing metadata.json, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(emptyDir, "tasks")); statErr == nil {
		t.Error("tasks directory should not have been created in an invalid module directory")
	}
}

// TestNewTask_RejectsCorruptMetadata verifies that malformed JSON in
// metadata.json is caught rather than silently ignored.
func TestNewTask_RejectsCorruptMetadata(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte("{{{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	err := NewTask(ComponentOptions{
		Name:        "mytask",
		WorkDir:     dir,
		TemplateDir: makeTaskTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for corrupt metadata.json, got nil")
	}
}

// TestNewTask_NameValidation covers the full range of invalid name inputs
// against the [a-z][a-z0-9_]* pattern, plus valid edge cases.
func TestNewTask_NameValidation(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "mytask", false},
		{"valid with numbers", "task2", false},
		{"valid with underscores", "my_task", false},
		{"valid with mixed", "my_task_2", false},
		{"init", "init", false},
		{"empty", "", true},
		{"uppercase start", "MyTask", true},
		{"all uppercase", "MYTASK", true},
		{"starts with digit", "2task", true},
		{"starts with underscore", "_task", true},
		{"starts with hyphen", "-task", true},
		{"contains double colon", "my::task", true},
		{"contains colon", "my:task", true},
		{"contains hyphen", "my-task", true},
		{"contains dot", "my.task", true},
		{"contains space", "my task", true},
		{"path separator slash", "my/task", true},
		{"path separator backslash", `my\task`, true},
		{"dot dot", "..", true},
		{"single dot", ".", true},
		{"unicode", "tâche", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := makeModuleDir(t, "myuser", "mymodule")
			err := NewTask(ComponentOptions{
				Name:        tc.input,
				WorkDir:     dir,
				TemplateDir: makeTaskTemplateDir(t),
			})
			if tc.wantErr && err == nil {
				t.Errorf("expected error for name %q, got nil", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for name %q: %v", tc.input, err)
			}
		})
	}
}

// TestNewTask_RefusesIfScriptFileExists_NoMetadataCreated verifies that a
// pre-existing .sh file causes an early error and the .json file is not created
// as a side effect, leaving no partial state on disk.
func TestNewTask_RefusesIfScriptFileExists_NoMetadataCreated(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	taskDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "mytask.sh"), []byte("# existing"), 0644); err != nil {
		t.Fatal(err)
	}

	err := NewTask(ComponentOptions{
		Name:        "mytask",
		WorkDir:     dir,
		TemplateDir: makeTaskTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for existing task script, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(taskDir, "mytask.json")); statErr == nil {
		t.Error("metadata file should not have been created when script file already exists")
	}
}

// TestNewTask_RefusesIfMetadataFileExists_NoScriptCreated verifies the inverse:
// a pre-existing .json file causes an error and the .sh file is not written.
// This guards against partial state when only the metadata was left behind.
func TestNewTask_RefusesIfMetadataFileExists_NoScriptCreated(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	taskDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "mytask.json"), []byte(`{"description":""}`), 0644); err != nil {
		t.Fatal(err)
	}

	err := NewTask(ComponentOptions{
		Name:        "mytask",
		WorkDir:     dir,
		TemplateDir: makeTaskTemplateDir(t),
	})
	if err == nil {
		t.Error("expected error for existing task metadata, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(taskDir, "mytask.sh")); statErr == nil {
		t.Error("script file should not have been created when metadata file already exists")
	}
}

// TestNewTask_ExistingScriptFileIsUntouched verifies that the content of a
// pre-existing .sh file is not modified when NewTask returns an error.
func TestNewTask_ExistingScriptFileIsUntouched(t *testing.T) {
	dir := makeModuleDir(t, "myuser", "mymodule")
	taskDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatal(err)
	}
	original := []byte("# do not touch")
	if err := os.WriteFile(filepath.Join(taskDir, "mytask.sh"), original, 0644); err != nil {
		t.Fatal(err)
	}

	_ = NewTask(ComponentOptions{
		Name:        "mytask",
		WorkDir:     dir,
		TemplateDir: makeTaskTemplateDir(t),
	})

	content, err := os.ReadFile(filepath.Join(taskDir, "mytask.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(original) {
		t.Errorf("existing script file was modified: got %q, want %q", string(content), string(original))
	}
}
