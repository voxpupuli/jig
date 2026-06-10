// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func NewFact(opts ComponentOptions) error {
	if err := validateComponentName(opts.Name); err != nil {
		return fmt.Errorf("invalid fact name: %w", err)
	}

	if strings.Contains(opts.Name, "::") {
		return fmt.Errorf("fact name cannot contain '::'")
	}

	_, err := GetMetadata(opts.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	factFileName := filepath.Join(opts.WorkDir, "lib", "facter", opts.Name+".rb")
	if _, err := os.Stat(factFileName); err == nil {
		return fmt.Errorf("fact %s already exists: %s", opts.Name, factFileName)
	}

	factTestFileName := filepath.Join(opts.WorkDir, "spec", "unit", "facter", opts.Name+"_spec.rb")
	if _, err := os.Stat(factTestFileName); err == nil {
		return fmt.Errorf("fact %s test already exists: %s", opts.Name, factTestFileName)
	}

	fmt.Printf("creating fact %s...\n", opts.Name)
	renderer := newRenderer(opts.TemplateDir)

	templates := []TemplateFile{
		{FileName: "fact/fact.rb", Destination: factFileName},
		{FileName: "fact/fact_spec.rb", Destination: factTestFileName},
	}

	data := struct{ Name string }{Name: opts.Name}

	if err := RenderTemplates(renderer, templates, data, false); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	return nil
}
