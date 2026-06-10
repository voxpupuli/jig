// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

func NewFunction(opts ComponentOptions) error {
	// Get metadata, we need the module name to construct the function name
	metadata, err := GetMetadata(opts.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	moduleName := metadata.ModuleName()
	functionName := fmt.Sprintf("%s::%s", moduleName, opts.Name)

	functionFile, err := ConstructDestinationFilename(
		opts.Name,
		moduleName,
		filepath.Join(opts.WorkDir, "functions"),
		".pp",
	)
	if err != nil {
		return fmt.Errorf("failed to construct function file path: %w", err)
	}
	if _, err := os.Stat(functionFile); err == nil {
		return fmt.Errorf("function %s already exists: %s", functionName, functionFile)
	}

	specFile, err := ConstructDestinationFilename(
		opts.Name,
		moduleName,
		filepath.Join(opts.WorkDir, "spec", "functions"),
		"_spec.rb",
	)
	if err != nil {
		return fmt.Errorf("failed to construct function test file path: %w", err)
	}
	if _, err := os.Stat(specFile); err == nil {
		return fmt.Errorf("function %s test already exists: %s", functionName, specFile)
	}

	renderer := newRenderer(opts.TemplateDir)

	templates := []TemplateFile{
		{FileName: "function/function.pp", Destination: functionFile},
		{FileName: "function/function_spec.rb", Destination: specFile},
	}

	data := struct{ Name string }{Name: functionName}

	fmt.Printf("creating function %s...\n", functionName)

	if err := RenderTemplates(renderer, templates, data, false); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	return nil
}
