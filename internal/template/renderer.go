// SPDX-License-Identifier: GPL-3.0-or-later
package template

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	tmplpkg "text/template"
)

//go:embed templates
var embeddedTemplates embed.FS

type Renderer struct {
	externalDir string
}

func NewRenderer() *Renderer {
	return &Renderer{}
}

func NewRendererWithExternalDir(dir string) *Renderer {
	return &Renderer{
		externalDir: dir,
	}
}

func (r Renderer) Render(templateName string, data any) (string, error) {
	if templateName == "" {
		return "", fmt.Errorf("template name cannot be empty")
	}

	// Prevent path traversal when reading from the external directory.
	// We check the embedded path too for consistency -- embed.FS would
	// reject ".." internally, but rejecting it here gives a clearer error.
	cleaned := filepath.Clean(templateName)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("invalid template name %q: must be a relative path within the template directory", templateName)
	}

	var content []byte
	var err error

	if r.externalDir != "" {
		externalPath := filepath.Join(r.externalDir, cleaned)
		// Double-check the joined path stays within the external dir.
		if !strings.HasPrefix(externalPath, filepath.Clean(r.externalDir)+string(filepath.Separator)) {
			return "", fmt.Errorf("invalid template name %q: resolves outside template directory", templateName)
		}

		content, err = os.ReadFile(externalPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("failed to read external template %s: %w", templateName, err)
			}
			// file not found in external dir, fall through to embedded templates
			content, err = embeddedTemplates.ReadFile("templates/" + cleaned)
			if err != nil {
				return "", fmt.Errorf("failed to read embedded template %s: %w", templateName, err)
			}
		}
	} else {
		content, err = embeddedTemplates.ReadFile("templates/" + cleaned)
		if err != nil {
			return "", fmt.Errorf("failed to read embedded template %s: %w", templateName, err)
		}
	}

	funcMap := tmplpkg.FuncMap{
		"upperFirst": func(s string) string {
			if s == "" {
				return ""
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
		"pascalCase": func(s string) string {
			parts := strings.Split(s, "_")
			for i, p := range parts {
				if p != "" {
					parts[i] = strings.ToUpper(p[:1]) + p[1:]
				}
			}
			return strings.Join(parts, "")
		},
	}

	t, err := tmplpkg.New(templateName).Funcs(funcMap).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to render template %s: %w", templateName, err)
	}
	return buf.String(), nil
}

func DumpTemplates(dest string) error {
	return fs.WalkDir(embeddedTemplates, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Strip the "templates/" prefix to get the relative path
		relPath := strings.TrimPrefix(path, "templates/")
		if relPath == "" {
			return nil
		}

		destPath := filepath.Join(dest, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		content, err := embeddedTemplates.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded template %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
		}

		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", destPath, err)
		}

		fmt.Printf("wrote %s\n", destPath)
		return nil
	})
}
