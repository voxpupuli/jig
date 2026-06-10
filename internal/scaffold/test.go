// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func NewTest(opts ComponentOptions) error {
	metadata, err := GetMetadata(opts.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	moduleName := metadata.ModuleName()
	var fqName string
	if opts.Name == "init" {
		fqName = moduleName
	} else {
		fqName = fmt.Sprintf("%s::%s", moduleName, opts.Name)
	}

	manifestFile, err := ConstructDestinationFilename(
		opts.Name,
		moduleName,
		filepath.Join(opts.WorkDir, "manifests"),
		".pp",
	)
	if err != nil {
		return fmt.Errorf("failed to construct manifest path: %w", err)
	}

	content, err := os.ReadFile(manifestFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no manifest found for %s: expected %s", fqName, manifestFile)
		}
		return fmt.Errorf("failed to read manifest %s: %w", manifestFile, err)
	}

	src := string(content)
	classPattern := regexp.MustCompile(`(?m)\bclass\s+` + regexp.QuoteMeta(fqName) + `[\s{(]`)
	definePattern := regexp.MustCompile(`(?m)\bdefine\s+` + regexp.QuoteMeta(fqName) + `[\s{(]`)

	var specDir, templateFile string
	switch {
	case classPattern.MatchString(src):
		specDir = filepath.Join(opts.WorkDir, "spec", "classes")
		templateFile = "class/class_spec.rb"
	case definePattern.MatchString(src):
		specDir = filepath.Join(opts.WorkDir, "spec", "defines")
		templateFile = "type/defined_type_spec.rb"
	default:
		return fmt.Errorf("no class or defined type named %s found in %s", fqName, manifestFile)
	}

	specFile, err := ConstructDestinationFilename(
		opts.Name,
		moduleName,
		specDir,
		"_spec.rb",
	)
	if err != nil {
		return fmt.Errorf("failed to construct spec file path: %w", err)
	}

	if _, err := os.Stat(specFile); err == nil {
		return fmt.Errorf("test for %s already exists: %s", fqName, specFile)
	}

	renderer := newRenderer(opts.TemplateDir)

	templates := []TemplateFile{
		{FileName: templateFile, Destination: specFile},
	}

	data := struct{ Name string }{Name: fqName}

	fmt.Printf("creating test for %s...\n", fqName)

	if err := RenderTemplates(renderer, templates, data, false); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	return nil
}
