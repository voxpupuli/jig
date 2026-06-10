// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func NewProvider(opts ComponentOptions) error {
	// Validate the provider name
	if err := validateComponentName(opts.Name); err != nil {
		return fmt.Errorf("invalid provider name: %w", err)
	}

	// Ensure the provider name matches the allowed pattern (lowercase letters,
	// numbers, and underscores)
	providerNamePattern := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	if !providerNamePattern.MatchString(opts.Name) {
		return fmt.Errorf(
			"provider name %q does not match the allowed pattern (%s)",
			opts.Name,
			providerNamePattern.String(),
		)
	}

	// Get the metadata to ensure we're in a valid module directory
	_, err := GetMetadata(opts.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	// Construct the file names and check for existing files
	providerFileName := filepath.Join(opts.WorkDir, "lib", "puppet", "provider", opts.Name, opts.Name+".rb")
	if _, err := os.Stat(providerFileName); err == nil {
		return fmt.Errorf("provider %s already exists: %s", opts.Name, providerFileName)
	}

	providerTestFileName := filepath.Join(opts.WorkDir, "spec", "unit", "puppet", "provider", opts.Name, opts.Name+"_spec.rb")
	if _, err := os.Stat(providerTestFileName); err == nil {
		return fmt.Errorf("provider %s test already exists: %s", opts.Name, providerTestFileName)
	}

	typeFileName := filepath.Join(opts.WorkDir, "lib", "puppet", "type", opts.Name+".rb")
	if _, err := os.Stat(typeFileName); err == nil {
		return fmt.Errorf("type %s already exists: %s", opts.Name, typeFileName)
	}

	typeTestFileName := filepath.Join(opts.WorkDir, "spec", "unit", "puppet", "type", opts.Name+"_spec.rb")
	if _, err := os.Stat(typeTestFileName); err == nil {
		return fmt.Errorf("type %s test already exists: %s", opts.Name, typeTestFileName)
	}

	templates := []TemplateFile{
		{FileName: "provider/provider.rb", Destination: providerFileName},
		{FileName: "provider/provider_spec.rb", Destination: providerTestFileName},
		{FileName: "provider/type.rb", Destination: typeFileName},
		{FileName: "provider/type_spec.rb", Destination: typeTestFileName},
	}

	fmt.Printf("creating provider %s...\n", opts.Name)

	renderer := newRenderer(opts.TemplateDir)
	data := struct{ Name string }{Name: opts.Name}

	if err := RenderTemplates(renderer, templates, data, false); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	return nil
}
