// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/voxpupuli/jig/internal/module"
	"github.com/voxpupuli/jig/internal/template"
)

// Renderer is the interface satisfied by *template.Renderer. Declared here
// so that RenderTemplates can be tested with a fake implementation.
type Renderer interface {
	Render(templateName string, data any) (string, error)
	ListTree(root string) ([]string, error)
}

type Options struct {
	ForgeUser   string
	Name        string
	Author      string
	License     string
	Summary     string
	Source      string
	Force       bool
	TargetDir   string
	TemplateDir string
	// TemplateURL, TemplateRef, and TemplateCommit describe the remote
	// template repository TemplateDir was fetched from, if any. They are
	// recorded in the generated metadata.json.
	TemplateURL    string
	TemplateRef    string
	TemplateCommit string
}

// moduleTemplateData is the data every template in the module tree renders
// with, shared by NewModule and Renew.
type moduleTemplateData struct {
	ModuleName string
	ForgeUser  string
	Author     string
	License    string
	ClassName  string
}

type ComponentOptions struct {
	Name        string
	TemplateDir string
	WorkDir     string
}

type TemplateFile struct {
	FileName    string
	Destination string
}

func newRenderer(templateDir string) Renderer {
	if templateDir != "" {
		return template.NewRendererWithExternalDir(templateDir)
	}
	return template.NewRenderer()
}

func BackupDir(path string) error {
	backupName := fmt.Sprintf("%s.bak.%s", path, time.Now().Format("20060102150405"))
	return os.Rename(path, backupName)
}

func GetMetadata(dir string) (module.Metadata, error) {
	if _, err := os.Stat(filepath.Join(dir, "metadata.json")); err != nil {
		return module.Metadata{}, fmt.Errorf("%s is not a valid module directory", dir)
	}

	metadata, err := module.ReadMetadata(filepath.Join(dir, "metadata.json"))
	if err != nil {
		return module.Metadata{}, fmt.Errorf("failed to read module metadata: %w", err)
	}

	return metadata, nil
}

func ConstructDestinationFilename(name string, moduleName string, prefix string, suffix string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}

	parts := strings.Split(name, "::")

	for _, part := range parts {
		if part == "" {
			return "", fmt.Errorf("name %q contains an empty component (check for leading, trailing, or consecutive '::')", name)
		}
		if strings.ContainsAny(part, "/\\") {
			return "", fmt.Errorf("name %q contains an invalid path separator in component %q", name, part)
		}
		if part == ".." || part == "." {
			return "", fmt.Errorf("name %q contains an invalid component %q", name, part)
		}
	}

	if parts[0] == moduleName {
		return "", fmt.Errorf("module name should not be included in class name")
	}

	fileName := parts[len(parts)-1]
	filePath := parts[:len(parts)-1]

	pathParts := append([]string{prefix}, filePath...)
	pathParts = append(pathParts, fileName+suffix)
	return filepath.Join(pathParts...), nil
}

func validateComponentName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("name %q contains an invalid path separator", name)
	}
	if name == ".." || name == "." {
		return fmt.Errorf("name %q is not a valid component name", name)
	}
	// Guard against names that would escape the target directory when joined.
	// filepath.Join cleans the path, so "foo/../bar" becomes "bar" -- we catch
	// this by checking the cleaned result stays within a known base.
	cleaned := filepath.Join("base", name)
	if !strings.HasPrefix(cleaned, filepath.Join("base", "")) {
		return fmt.Errorf("name %q would escape the target directory", name)
	}
	return nil
}

func RenderTemplates(renderer Renderer, templateFiles []TemplateFile, data any, overwrite bool) error {
	for _, t := range templateFiles {
		if !overwrite {
			if _, err := os.Stat(t.Destination); err == nil {
				return fmt.Errorf("file %s already exists", t.Destination)
			}
		}
		rendered, err := renderer.Render(t.FileName, data)
		if err != nil {
			return fmt.Errorf("failed to render template %s: %w", t.FileName, err)
		}
		if err := os.MkdirAll(filepath.Dir(t.Destination), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(t.Destination), err)
		}
		if err := os.WriteFile(t.Destination, []byte(rendered), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", t.Destination, err)
		}
	}
	return nil
}
