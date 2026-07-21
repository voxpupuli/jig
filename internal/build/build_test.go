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

	"github.com/voxpupuli/jig/internal/config"
)

// makeBuildDir creates a minimal but realistic module directory suitable for
// DoBuild. It writes metadata.json and a small set of files that mirror what
// jig new module produces: some allowed by the spec allowlist, some not.
func makeBuildDir(t *testing.T, forgeUser, moduleName string) string {
	t.Helper()
	dir := t.TempDir()

	writeTestMetadata(t, dir, forgeUser, moduleName, "0.1.0")

	// Files the spec allowlist permits
	writeFile(t, dir, "manifests/init.pp", "class "+moduleName+" {}")
	writeFile(t, dir, "README.md", "# "+moduleName)
	writeFile(t, dir, "CHANGELOG.md", "## Release 0.1.0")
	writeFile(t, dir, "hiera.yaml", "---\nversion: 5")
	writeFile(t, dir, "data/common.yaml", "---")
	writeFile(t, dir, "lib/facter/custom.rb", "# custom fact")

	// Files it does not
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

// writeTestMetadata writes a minimal valid metadata.json into dir.
func writeTestMetadata(t *testing.T, dir, forgeUser, moduleName, version string) {
	t.Helper()
	meta := map[string]any{
		"name":                    forgeUser + "-" + moduleName,
		"version":                 version,
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
	metaData, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "metadata.json", string(metaData))
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

// Without any jig.toml, the build must apply the spec allowlist: files the
// specification permits go in, everything else stays out. No ignore file of
// any kind is consulted.
func TestDoBuild_DefaultSpecAllowlist(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")

	if err := DoBuild(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := archiveEntries(t, filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz"))

	included := []string{
		"metadata.json",
		"manifests/init.pp",
		"README.md",
		"CHANGELOG.md",
		"hiera.yaml",
		"data/common.yaml",
		"lib/facter/custom.rb",
	}
	for _, suffix := range included {
		if !containsEntry(entries, suffix) {
			t.Errorf("expected archive to contain %q, but it did not. entries: %v", suffix, entries)
		}
	}

	excluded := []string{
		"Gemfile",
		"Rakefile",
		".gitignore",
		".rubocop.yml",
		".gitkeep",
		"spec/spec_helper.rb",
	}
	for _, suffix := range excluded {
		if containsEntry(entries, suffix) {
			t.Errorf("expected %q to be excluded by the spec allowlist, but it was in the archive", suffix)
		}
	}
}

// In deny mode (the default), configured exceptions extend the spec
// allowlist rather than replacing it.
func TestDoBuild_DenyExceptionsExtendSpec(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")
	writeFile(t, dir, "extra.txt", "shipped on purpose")
	writeFile(t, dir, "jig.toml", "[build]\nexceptions = [\"/extra.txt\"]\n")

	if err := DoBuild(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := archiveEntries(t, filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz"))
	if !containsEntry(entries, "extra.txt") {
		t.Error("extra.txt should have been included by the configured exception")
	}
	if !containsEntry(entries, "manifests/init.pp") {
		t.Error("manifests/init.pp should still be included: exceptions must extend the spec allowlist, not replace it")
	}
	if containsEntry(entries, "Gemfile") {
		t.Error("Gemfile should still be excluded in deny mode")
	}
}

// action = "allow" packages everything except the exceptions -- the old
// denylist workflow, now driven from jig.toml.
func TestDoBuild_AllowActionExcludesExceptions(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")
	writeFile(t, dir, "random.txt", "not in the spec allowlist")
	writeFile(t, dir, "jig.toml", "[build]\naction = \"allow\"\nexceptions = [\"/Gemfile\", \"/spec/**\"]\n")

	if err := DoBuild(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := archiveEntries(t, filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz"))
	if !containsEntry(entries, "random.txt") {
		t.Error("random.txt should have been included: allow mode packages everything not excepted")
	}
	if containsEntry(entries, "Gemfile") {
		t.Error("Gemfile should have been excluded by the configured exception")
	}
	if containsEntry(entries, "spec/spec_helper.rb") {
		t.Error("spec/spec_helper.rb should have been excluded by the configured exception")
	}
	if containsEntry(entries, ".gitkeep") {
		t.Error(".gitkeep markers must be excluded in every mode")
	}
	if containsEntry(entries, "jig.toml") {
		t.Error("jig.toml must never be packaged")
	}
}

func TestDoBuild_JigTomlNeverInArchive(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")
	writeFile(t, dir, "jig.toml", "")

	if err := DoBuild(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := archiveEntries(t, filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz"))
	if containsEntry(entries, "jig.toml") {
		t.Error("jig.toml must never be packaged")
	}
}

func TestDoBuild_InvalidBuildAction(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")
	writeFile(t, dir, "jig.toml", "[build]\naction = \"denylist\"\n")

	err := DoBuild(dir)
	if err == nil {
		t.Fatal("expected error for invalid build action, got nil")
	}
	if !strings.Contains(err.Error(), "denylist") {
		t.Errorf("error %q does not mention the invalid action", err.Error())
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
			writeTestMetadata(t, dir, tc.forgeUser, tc.moduleName, tc.version)

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

// A leftover ignore file must produce a warning naming it, since builds no
// longer obey it. .gitignore belongs to git and must stay silent.
func TestDoBuild_WarnsAboutIgnoreFiles(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")
	writeFile(t, dir, ".pdkignore", "/spec/\n")
	writeFile(t, dir, ".pmtignore", "/spec/\n")

	out := captureStdout(t, func() {
		if err := DoBuild(dir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, name := range []string{".pdkignore", ".pmtignore"} {
		if !strings.Contains(out, name) || !strings.Contains(out, "not obeyed") {
			t.Errorf("expected a warning naming %s, got output: %q", name, out)
		}
	}
	if strings.Contains(out, ".gitignore") {
		t.Errorf(".gitignore must not be warned about, got output: %q", out)
	}

	// The ignore file must also not change what gets packaged.
	entries := archiveEntries(t, filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz"))
	if !containsEntry(entries, "manifests/init.pp") {
		t.Error("manifests/init.pp should be included regardless of ignore files")
	}
}

func TestFindIgnoreFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".pdkignore", "")
	writeFile(t, dir, ".pmtignore", "")
	writeFile(t, dir, ".customignore", "")
	writeFile(t, dir, ".gitignore", "")                                      // exempt: belongs to git
	writeFile(t, dir, "notignore", "")                                       // no leading dot
	writeFile(t, dir, ".ignorefile", "")                                     // wrong suffix
	writeFile(t, dir, "sub/.pdkignore", "")                                  // not in the module root
	if err := os.Mkdir(filepath.Join(dir, ".dirignore"), 0755); err != nil { // directories don't count
		t.Fatal(err)
	}

	found := findIgnoreFiles(dir)
	var names []string
	for _, f := range found {
		names = append(names, filepath.Base(f))
	}

	want := []string{".customignore", ".pdkignore", ".pmtignore"}
	if len(names) != len(want) {
		t.Fatalf("found %v, want %v", names, want)
	}
	for i, name := range want {
		if names[i] != name {
			t.Errorf("found %v, want %v", names, want)
			break
		}
	}
}

// --- buildFilter ---

func TestBuildFilter_DefaultDeny(t *testing.T) {
	filter, err := newBuildFilter(config.BuildConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := []struct {
		path     string
		included bool
	}{
		{"metadata.json", true},
		{"CHANGELOG.md", true},
		{"README.md", true},
		{"REFERENCE.md", true},
		{"LICENSE", true},
		{"hiera.yaml", true},
		{"manifests/init.pp", true},
		{"manifests/sub/deep.pp", true},
		{"lib/facter/custom.rb", true},
		{"data/common.yaml", true},
		{"Gemfile", false},
		{"Rakefile", false},
		{".gitignore", false},
		{".rubocop.yml", false},
		{"spec/spec_helper.rb", false},
		{"data/.gitkeep", false},
		{"jig.toml", false},
		{"pkg/myuser-mymodule-0.1.0.tar.gz", false},
	}
	for _, tc := range cases {
		if got := filter.Include(tc.path); got != tc.included {
			t.Errorf("Include(%q): got %v, want %v", tc.path, got, tc.included)
		}
	}
}

func TestBuildFilter_AllowMode(t *testing.T) {
	filter, err := newBuildFilter(config.BuildConfig{
		Action:     config.BuildActionAllow,
		Exceptions: []string{"/Gemfile", "/spec/**"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := []struct {
		path     string
		included bool
	}{
		{"random.txt", true},
		{".gitignore", true},
		{"Gemfile", false},
		{"spec/spec_helper.rb", false},
		{"jig.toml", false},
		{"data/.gitkeep", false},
		{"pkg/archive.tar.gz", false},
	}
	for _, tc := range cases {
		if got := filter.Include(tc.path); got != tc.included {
			t.Errorf("Include(%q): got %v, want %v", tc.path, got, tc.included)
		}
	}
}

func TestBuildFilter_InvalidAction(t *testing.T) {
	if _, err := newBuildFilter(config.BuildConfig{Action: "bogus"}); err == nil {
		t.Fatal("expected error for invalid action, got nil")
	}
}
