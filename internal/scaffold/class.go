// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

func NewClass(opts ComponentOptions) error {
	metadata, err := GetMetadata(opts.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	moduleName := metadata.ModuleName()

	classFile, err := ConstructDestinationFilename(
		opts.Name,
		moduleName,
		filepath.Join(opts.WorkDir, "manifests"),
		".pp",
	)
	if err != nil {
		return fmt.Errorf("failed to construct class file path: %w", err)
	}

	specFile, err := ConstructDestinationFilename(
		opts.Name,
		moduleName,
		filepath.Join(opts.WorkDir, "spec", "classes"),
		"_spec.rb",
	)
	if err != nil {
		return fmt.Errorf("failed to construct spec file path: %w", err)
	}

	className := fmt.Sprintf("%s::%s", moduleName, opts.Name)

	if _, err := os.Stat(classFile); err == nil {
		return fmt.Errorf("class %s already exists: %s", className, classFile)
	}

	renderer := newRenderer(opts.TemplateDir)

	templates := []TemplateFile{
		{FileName: "class/class.pp", Destination: classFile},
		{FileName: "class/class_spec.rb", Destination: specFile},
	}

	data := struct{ Name string }{Name: className}

	fmt.Printf("creating class %s...\n", className)

	if err := RenderTemplates(renderer, templates, data, false); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	return nil
}
