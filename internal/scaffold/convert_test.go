package scaffold

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/voxpupuli/jig/v2/internal/config"
	"github.com/voxpupuli/jig/v2/internal/module"
)

func TestConvertModule(t *testing.T) {
	tmpDir := t.TempDir()

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	err := os.WriteFile(metadataPath, []byte(`{"name": "test-module", "version": "0.1.0", "author": "a", "license": "l", "summary": "s", "source": "https://example.com", "dependencies": [], "requirements": [], "operatingsystem_support": [], "tags": []}`), 0644)
	if err != nil {
		t.Fatalf("Error while generating the mock metadata.json: %v", err)
	}

	err = ConvertModule(ConvertOptions{TargetDir: tmpDir})
	if err != nil {
		t.Fatalf("ConvertModule failed unexpected: %v", err)
	}

	expectedFiles := []string{
		"Gemfile",
		"Rakefile",
		filepath.Join("spec", "spec_helper.rb"),
	}

	for _, file := range expectedFiles {
		fullPath := filepath.Join(tmpDir, file)

		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			t.Errorf("Expected file was not created: %s", file)
			continue
		}
		if err != nil {
			t.Errorf("Error while checking file %s: %v", file, err)
			continue
		}

		if info.Size() == 0 {
			t.Errorf("File %s was created, but empty", file)
		}
	}
}

func TestConvertModule_MissingMetadata_CreatesIt(t *testing.T) {
	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "puppet-nftables")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	err := ConvertModule(ConvertOptions{
		TargetDir: moduleDir,
		ForgeUser: "puppet",
		Author:    "Jane Doe",
		License:   "Apache-2.0",
		Summary:   "manages nftables",
		Source:    "https://example.com/nftables",
	})
	if err != nil {
		t.Fatalf("ConvertModule failed unexpectedly: %v", err)
	}

	meta, err := module.ReadMetadata(filepath.Join(moduleDir, "metadata.json"))
	if err != nil {
		t.Fatalf("failed to read generated metadata.json: %v", err)
	}
	if meta.Name != "puppet-nftables" {
		t.Errorf("expected name %q, got %q", "puppet-nftables", meta.Name)
	}
	if meta.Version != "0.1.0" {
		t.Errorf("expected default version, got %q", meta.Version)
	}

	if _, err := os.Stat(filepath.Join(moduleDir, config.ModuleConfigFileName)); err != nil {
		t.Errorf("expected jig.toml to be created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(moduleDir, "Gemfile")); err != nil {
		t.Errorf("expected Gemfile to be created: %v", err)
	}
}

func TestConvertModule_MissingMetadata_UsesModulefile(t *testing.T) {
	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "whatever-name")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	modulefile := `name    'binford2k-demo'
version '2.3.4'
author  'Binford Tools'
license 'MIT'
summary 'A demo module'
dependency 'puppetlabs/stdlib', '>= 3.2.0 < 5.0.0'
`
	if err := os.WriteFile(filepath.Join(moduleDir, "Modulefile"), []byte(modulefile), 0644); err != nil {
		t.Fatalf("failed to write Modulefile: %v", err)
	}

	mf, err := ParseModulefile(filepath.Join(moduleDir, "Modulefile"))
	if err != nil {
		t.Fatalf("ParseModulefile failed: %v", err)
	}
	forgeUser, name := SplitForgeName(mf.Name)

	err = ConvertModule(ConvertOptions{
		TargetDir:     moduleDir,
		ForgeUser:     forgeUser,
		Name:          name,
		Author:        mf.Author,
		License:       mf.License,
		Summary:       mf.Summary,
		Version:       mf.Version,
		Dependencies:  mf.Dependencies,
		HasModulefile: true,
	})
	if err != nil {
		t.Fatalf("ConvertModule failed unexpectedly: %v", err)
	}

	meta, err := module.ReadMetadata(filepath.Join(moduleDir, "metadata.json"))
	if err != nil {
		t.Fatalf("failed to read generated metadata.json: %v", err)
	}
	if meta.Name != "binford2k-demo" {
		t.Errorf("expected name %q, got %q", "binford2k-demo", meta.Name)
	}
	if meta.Version != "2.3.4" {
		t.Errorf("expected version %q, got %q", "2.3.4", meta.Version)
	}
	if meta.Author != "Binford Tools" {
		t.Errorf("expected author %q, got %q", "Binford Tools", meta.Author)
	}
	if len(meta.Dependencies) != 1 || meta.Dependencies[0].Name != "puppetlabs-stdlib" {
		t.Errorf("expected one dependency on puppetlabs-stdlib, got %+v", meta.Dependencies)
	}

	if _, err := os.Stat(filepath.Join(moduleDir, "Modulefile")); err != nil {
		t.Errorf("Modulefile should be left in place: %v", err)
	}
}

func TestConvertModule_InvalidJSON_Stops(t *testing.T) {
	tmpDir := t.TempDir()
	metadataPath := filepath.Join(tmpDir, "metadata.json")
	if err := os.WriteFile(metadataPath, []byte(`{not valid json`), 0644); err != nil {
		t.Fatalf("failed to write broken metadata.json: %v", err)
	}

	err := ConvertModule(ConvertOptions{TargetDir: tmpDir})
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got none")
	}

	if _, statErr := os.Stat(filepath.Join(tmpDir, "Gemfile")); !os.IsNotExist(statErr) {
		t.Error("Gemfile should not be written when metadata.json fails to parse")
	}
}

func TestConvertModule_RepairsPartialMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	metadataPath := filepath.Join(tmpDir, "metadata.json")
	if err := os.WriteFile(metadataPath, []byte(`{"name": "test-module", "author": "a", "license": "l", "summary": "s", "source": "https://example.com"}`), 0644); err != nil {
		t.Fatalf("failed to write partial metadata.json: %v", err)
	}

	if err := ConvertModule(ConvertOptions{TargetDir: tmpDir}); err != nil {
		t.Fatalf("ConvertModule failed unexpectedly: %v", err)
	}

	meta, err := module.ReadMetadata(metadataPath)
	if err != nil {
		t.Fatalf("failed to read repaired metadata.json: %v", err)
	}
	if meta.Version != "0.1.0" {
		t.Errorf("expected version to default to 0.1.0, got %q", meta.Version)
	}
	if meta.Dependencies == nil {
		t.Error("expected dependencies to default to an empty list, not nil")
	}
	if meta.Author != "a" {
		t.Errorf("existing author should not be overwritten, got %q", meta.Author)
	}

	before, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("failed to read metadata.json: %v", err)
	}

	if err := ConvertModule(ConvertOptions{TargetDir: tmpDir}); err != nil {
		t.Fatalf("second ConvertModule run failed: %v", err)
	}
	after, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("failed to read metadata.json: %v", err)
	}
	if string(before) != string(after) {
		t.Error("running convert twice should be a no-op on metadata.json")
	}
}

// Fields jig doesn't model (a PDK-era pdk-version, say) must survive repair
// untouched, not be silently dropped when the file is rewritten.
func TestConvertModule_RepairPreservesUnknownKeys(t *testing.T) {
	tmpDir := t.TempDir()
	metadataPath := filepath.Join(tmpDir, "metadata.json")
	original := `{
  "name": "test-module",
  "author": "a",
  "license": "l",
  "summary": "s",
  "source": "https://example.com",
  "pdk-version": "3.0.0",
  "data_provider": "function"
}`
	if err := os.WriteFile(metadataPath, []byte(original), 0644); err != nil {
		t.Fatalf("failed to write metadata.json: %v", err)
	}

	if err := ConvertModule(ConvertOptions{TargetDir: tmpDir}); err != nil {
		t.Fatalf("ConvertModule failed unexpectedly: %v", err)
	}

	repaired, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("failed to read repaired metadata.json: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(repaired, &raw); err != nil {
		t.Fatalf("repaired metadata.json is not valid JSON: %v", err)
	}

	if v, ok := raw["pdk-version"]; !ok || strings.Trim(string(v), `"`) != "3.0.0" {
		t.Errorf("expected pdk-version to survive repair, got %v (present: %v)", v, ok)
	}
	if v, ok := raw["data_provider"]; !ok || strings.Trim(string(v), `"`) != "function" {
		t.Errorf("expected data_provider to survive repair, got %v (present: %v)", v, ok)
	}
	if _, ok := raw["version"]; !ok {
		t.Error("expected version to have been defaulted in")
	}
}

func TestConvertModule_DryRunChangesNothing(t *testing.T) {
	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "puppet-nftables")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	var out strings.Builder
	err := ConvertModule(ConvertOptions{
		TargetDir: moduleDir,
		ForgeUser: "puppet",
		Author:    "Jane Doe",
		License:   "Apache-2.0",
		DryRun:    true,
		Out:       &out,
	})
	if err != nil {
		t.Fatalf("ConvertModule failed unexpectedly: %v", err)
	}

	if _, err := os.Stat(filepath.Join(moduleDir, "metadata.json")); !os.IsNotExist(err) {
		t.Error("dry-run should not create metadata.json")
	}
	if _, err := os.Stat(filepath.Join(moduleDir, "Gemfile")); !os.IsNotExist(err) {
		t.Error("dry-run should not create Gemfile")
	}
	if !strings.Contains(out.String(), "would create") {
		t.Errorf("expected dry-run output to describe what would be created, got: %q", out.String())
	}
}
