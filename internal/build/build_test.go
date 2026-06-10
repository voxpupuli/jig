// SPDX-License-Identifier: GPL-3.0-or-later
package build

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeBuildDir creates a minimal but realistic module directory suitable for
// DoBuild. It writes metadata.json, a .pdkignore, and a small set of files
// that mirror what jig new module produces.
func makeBuildDir(t *testing.T, forgeUser, moduleName string) string {
	t.Helper()
	dir := t.TempDir()

	// Write metadata.json
	meta := map[string]any{
		"name":                    forgeUser + "-" + moduleName,
		"version":                 "0.1.0",
		"author":                  forgeUser,
		"license":                 "Apache-2.0",
		"summary":                 "A test module",
		"source":                  "https://example.com",
		"dependencies":            []any{},
		"requirements":            []any{},
		"operatingsystem_support": []any{},
		"tags":                    []any{},
		"pdk-version":             "3.4.0",
	}
	metaData, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "metadata.json", string(metaData))

	// Write a realistic .pdkignore
	pdkIgnore := strings.Join([]string{
		"/.git/",
		"/pkg/",
		"/spec/",
		"/Gemfile",
		"/Rakefile",
		"/.gitignore",
		"/.pdkignore",
		"/.rubocop.yml",
	}, "\n")
	writeFile(t, dir, ".pdkignore", pdkIgnore)

	// Files that should be included in the archive
	writeFile(t, dir, "manifests/init.pp", "class "+moduleName+" {}")
	writeFile(t, dir, "README.md", "# "+moduleName)
	writeFile(t, dir, "CHANGELOG.md", "## Release 0.1.0")
	writeFile(t, dir, "hiera.yaml", "---\nversion: 5")
	writeFile(t, dir, "data/common.yaml", "---")

	// Files that should be excluded
	writeFile(t, dir, "Gemfile", "source 'https://rubygems.org'")
	writeFile(t, dir, "Rakefile", "require 'bundler'")
	writeFile(t, dir, ".gitignore", "/pkg/")
	writeFile(t, dir, ".rubocop.yml", "---")
	writeFile(t, dir, "data/.gitkeep", "")
	writeFile(t, dir, "spec/spec_helper.rb", "require 'puppetlabs_spec_helper'")

	return dir
}

// writeFile creates a file at dir/relPath with the given content,
// creating any necessary parent directories.
func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	path := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// archiveEntries opens a tar.gz file and returns the list of entry names.
func archiveEntries(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open archive %s: %v", path, err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var entries []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("failed to read tar entry: %v", err)
		}
		entries = append(entries, hdr.Name)
	}
	return entries
}

// containsEntry checks whether entries contains a string with the given suffix.
func containsEntry(entries []string, suffix string) bool {
	for _, e := range entries {
		if strings.HasSuffix(filepath.ToSlash(e), suffix) {
			return true
		}
	}
	return false
}

// --- DoBuild ---

func TestDoBuild_CreatesArchive(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")

	if err := DoBuild(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	archivePath := filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz")
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("expected archive at %s: %v", archivePath, err)
	}
}

func TestDoBuild_ArchivePrefix(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")

	if err := DoBuild(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := archiveEntries(t, filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz"))
	if len(entries) == 0 {
		t.Fatal("archive is empty")
	}

	prefix := "myuser-mymodule-0.1.0/"
	for _, e := range entries {
		normalized := filepath.ToSlash(e)
		if !strings.HasPrefix(normalized, prefix) {
			t.Errorf("archive entry %q does not start with prefix %q", e, prefix)
		}
	}
}

func TestDoBuild_IncludedFiles(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")

	if err := DoBuild(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := archiveEntries(t, filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz"))

	expectedSuffixes := []string{
		"metadata.json",
		"manifests/init.pp",
		"README.md",
		"CHANGELOG.md",
		"hiera.yaml",
		"data/common.yaml",
	}
	for _, suffix := range expectedSuffixes {
		if !containsEntry(entries, suffix) {
			t.Errorf("expected archive to contain %q, but it did not. entries: %v", suffix, entries)
		}
	}
}

func TestDoBuild_ExcludedFiles(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")

	if err := DoBuild(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := archiveEntries(t, filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz"))

	// Excluded by .pdkignore
	pdkIgnoreExcluded := []string{
		"Gemfile",
		"Rakefile",
		".gitignore",
		".pdkignore",
		"spec/spec_helper.rb",
	}
	for _, suffix := range pdkIgnoreExcluded {
		if containsEntry(entries, suffix) {
			t.Errorf("expected %q to be excluded by .pdkignore, but it was in the archive", suffix)
		}
	}

	// Excluded by hardcoded patterns
	hardcodedExcluded := []string{
		".gitkeep",
		".rubocop.yml",
	}
	for _, suffix := range hardcodedExcluded {
		if containsEntry(entries, suffix) {
			t.Errorf("expected %q to be excluded by hardcoded pattern, but it was in the archive", suffix)
		}
	}
}

func TestDoBuild_PkgDirNotInArchive(t *testing.T) {
	// The pkg/ directory is created by DoBuild itself and should never
	// appear as an entry inside the archive it is writing into.
	dir := makeBuildDir(t, "myuser", "mymodule")

	if err := DoBuild(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := archiveEntries(t, filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz"))
	for _, e := range entries {
		if strings.Contains(filepath.ToSlash(e), "/pkg/") {
			t.Errorf("pkg/ directory or its contents found in archive: %q", e)
		}
	}
}

func TestDoBuild_MissingMetadata(t *testing.T) {
	dir := t.TempDir()
	// No metadata.json
	err := DoBuild(dir)
	if err == nil {
		t.Error("expected error for missing metadata.json, got nil")
	}
}

func TestDoBuild_InvalidMetadata(t *testing.T) {
	dir := t.TempDir()

	// Write metadata that will fail Validate -- missing summary and source
	meta := map[string]any{
		"name":                    "myuser-mymodule",
		"version":                 "0.1.0",
		"author":                  "myuser",
		"license":                 "Apache-2.0",
		"summary":                 "", // missing
		"source":                  "", // missing
		"dependencies":            []any{},
		"requirements":            []any{},
		"operatingsystem_support": []any{},
		"tags":                    []any{},
		"pdk-version":             "3.4.0",
	}
	metaData, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "metadata.json", string(metaData))

	err = DoBuild(dir)
	if err == nil {
		t.Error("expected error for invalid metadata, got nil")
	}
}

func TestDoBuild_MissingPdkIgnore(t *testing.T) {
	dir := t.TempDir()

	meta := map[string]any{
		"name":                    "myuser-mymodule",
		"version":                 "0.1.0",
		"author":                  "myuser",
		"license":                 "Apache-2.0",
		"summary":                 "test",
		"source":                  "https://example.com",
		"dependencies":            []any{},
		"requirements":            []any{},
		"operatingsystem_support": []any{},
		"tags":                    []any{},
		"pdk-version":             "3.4.0",
	}
	metaData, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "metadata.json", string(metaData))
	// No .pdkignore

	err = DoBuild(dir)
	if err == nil {
		t.Error("expected error for missing .pdkignore, got nil")
	}
}

func TestDoBuild_EmptyPdkIgnore(t *testing.T) {
	// An empty .pdkignore is valid -- no patterns means nothing is excluded
	// (except the hardcoded ones). The build should succeed.
	dir := t.TempDir()

	meta := map[string]any{
		"name":                    "myuser-mymodule",
		"version":                 "0.1.0",
		"author":                  "myuser",
		"license":                 "Apache-2.0",
		"summary":                 "test",
		"source":                  "https://example.com",
		"dependencies":            []any{},
		"requirements":            []any{},
		"operatingsystem_support": []any{},
		"tags":                    []any{},
		"pdk-version":             "3.4.0",
	}
	metaData, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "metadata.json", string(metaData))
	writeFile(t, dir, ".pdkignore", "")
	writeFile(t, dir, "manifests/init.pp", "class mymodule {}")

	if err := DoBuild(dir); err != nil {
		t.Fatalf("unexpected error for empty .pdkignore: %v", err)
	}
}

func TestDoBuild_PdkIgnoreCommentsAndBlanks(t *testing.T) {
	// A .pdkignore with only comments and blank lines should behave the
	// same as an empty one.
	dir := t.TempDir()

	meta := map[string]any{
		"name":                    "myuser-mymodule",
		"version":                 "0.1.0",
		"author":                  "myuser",
		"license":                 "Apache-2.0",
		"summary":                 "test",
		"source":                  "https://example.com",
		"dependencies":            []any{},
		"requirements":            []any{},
		"operatingsystem_support": []any{},
		"tags":                    []any{},
		"pdk-version":             "3.4.0",
	}
	metaData, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "metadata.json", string(metaData))
	writeFile(t, dir, ".pdkignore", "# this is a comment\n\n# another comment\n")
	writeFile(t, dir, "manifests/init.pp", "class mymodule {}")

	if err := DoBuild(dir); err != nil {
		t.Fatalf("unexpected error for comment-only .pdkignore: %v", err)
	}
}

func TestDoBuild_ArchiveName(t *testing.T) {
	// Verify the archive name format: forgeuser-modulename-version.tar.gz
	cases := []struct {
		forgeUser    string
		moduleName   string
		version      string
		expectedName string
	}{
		{"myuser", "mymodule", "0.1.0", "myuser-mymodule-0.1.0.tar.gz"},
		{"puppet", "apache", "1.2.3", "puppet-apache-1.2.3.tar.gz"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s-%s", tc.forgeUser, tc.moduleName), func(t *testing.T) {
			dir := makeBuildDir(t, tc.forgeUser, tc.moduleName)

			// Update version in metadata.json
			meta := map[string]any{
				"name":                    tc.forgeUser + "-" + tc.moduleName,
				"version":                 tc.version,
				"author":                  tc.forgeUser,
				"license":                 "Apache-2.0",
				"summary":                 "test",
				"source":                  "https://example.com",
				"dependencies":            []any{},
				"requirements":            []any{},
				"operatingsystem_support": []any{},
				"tags":                    []any{},
				"pdk-version":             "3.4.0",
			}
			metaData, err := json.Marshal(meta)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, dir, "metadata.json", string(metaData))

			if err := DoBuild(dir); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			archivePath := filepath.Join(dir, "pkg", tc.expectedName)
			if _, err := os.Stat(archivePath); err != nil {
				t.Errorf("expected archive at %s: %v", archivePath, err)
			}
		})
	}
}
