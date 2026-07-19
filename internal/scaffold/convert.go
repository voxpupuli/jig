package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/voxpupuli/jig/internal/template" 
)

func ConvertModule(targetDir string) error {
	if _, err := os.Stat(filepath.Join(targetDir, "metadata.json")); os.IsNotExist(err) {
		return fmt.Errorf("metadata.json not found; the convert command must be executed from the module base directory.")
	}

	filesToUpdate := []struct {
		TemplatePath string
		DestPath     string
	}{
		{TemplatePath: "module/Gemfile", DestPath: "Gemfile"},
		{TemplatePath: "module/Rakefile", DestPath: "Rakefile"},
		{TemplatePath: "module/spec/spec_helper.rb", DestPath: filepath.Join("spec", "spec_helper.rb")},
	}

	specDir := filepath.Join(targetDir, "spec")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		return fmt.Errorf("Error creating spec directory: %w", err)
	}

	renderer := template.NewRenderer() 

	for _, f := range filesToUpdate {
		fullDestPath := filepath.Join(targetDir, f.DestPath)

		content, err := renderer.Render(f.TemplatePath, nil)
		if err != nil {
			return fmt.Errorf("Error while parsing template %s: %w", f.TemplatePath, err)
		}

		err = os.WriteFile(fullDestPath, []byte(content), 0644)
		if err != nil {
			return fmt.Errorf("Error writing file %s: %w", fullDestPath, err)
		}
	}

	return nil
}
