package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConvertModule(t *testing.T) {
	tmpDir := t.TempDir()

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	err := os.WriteFile(metadataPath, []byte(`{"name": "test-module", "version": "0.1.0"}`), 0644)
	if err != nil {
		t.Fatalf("Error while generating the mock metadata.json: %v", err)
	}

	err = ConvertModule(tmpDir)
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

func TestConvertModule_MissingMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	err := ConvertModule(tmpDir)
	if err == nil {
		t.Error("Expected an error, as metadata.json is missing, but convertModule ran successfully")
	}
}
