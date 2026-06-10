// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

func NewDefinedType(opts ComponentOptions) error {
	metadata, err := GetMetadata(opts.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	moduleName := metadata.ModuleName()

	typeFile, err := ConstructDestinationFilename(
		opts.Name,
		moduleName,
		filepath.Join(opts.WorkDir, "manifests"),
		".pp",
	)
	if err != nil {
		return fmt.Errorf("failed to construct defined_type file path: %w", err)
	}

	specFile, err := ConstructDestinationFilename(
		opts.Name,
		moduleName,
		filepath.Join(opts.WorkDir, "spec", "defines"),
		"_spec.rb",
	)
	if err != nil {
		return fmt.Errorf("failed to construct defined_type test file path: %w", err)
	}

	typeName := fmt.Sprintf("%s::%s", moduleName, opts.Name)

	if _, err := os.Stat(typeFile); err == nil {
		return fmt.Errorf("defined_type %s already exists: %s", typeName, typeFile)
	}
	if _, err := os.Stat(specFile); err == nil {
		return fmt.Errorf("defined_type %s test already exists: %s", typeName, specFile)
	}

	renderer := newRenderer(opts.TemplateDir)

	templates := []TemplateFile{
		{FileName: "type/defined_type.pp", Destination: typeFile},
		{FileName: "type/defined_type_spec.rb", Destination: specFile},
	}

	data := struct{ Name string }{Name: typeName}

	fmt.Printf("creating defined_type %s...\n", typeName)

	if err := RenderTemplates(renderer, templates, data, false); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	return nil
}
