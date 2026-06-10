// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func NewTask(opts ComponentOptions) error {
	// Get the metadata, this is only done here to ensure we're in a valid
	// module directory.'
	_, err := GetMetadata(opts.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	// Validate that the task name meets the task name requirements
	// [a-z][a-z0-9_]*
	taskNamePattern := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	if !taskNamePattern.MatchString(opts.Name) {
		return fmt.Errorf(
			"task name %q does not match the allowed pattern (%s)",
			opts.Name,
			taskNamePattern.String(),
		)
	}

	// Check to see if a task by given name already exists in the module, and
	// return an error if it does.
	taskFileName := filepath.Join(opts.WorkDir, "tasks", opts.Name+".sh")
	if _, err := os.Stat(taskFileName); err == nil {
		return fmt.Errorf(
			"task %s already exists: %s",
			opts.Name,
			taskFileName,
		)
	}

	// Check to see if a task metadata file exists in the module, and return an
	// error if it does.
	taskMetadataName := filepath.Join(opts.WorkDir, "tasks", opts.Name+".json")
	if _, err := os.Stat(taskMetadataName); err == nil {
		return fmt.Errorf(
			"metadata file for task %s already exists: %s",
			opts.Name,
			taskMetadataName,
		)
	}

	renderer := newRenderer(opts.TemplateDir)

	templates := []TemplateFile{
		{FileName: "task/task.sh", Destination: taskFileName},
		{FileName: "task/metadata.json", Destination: taskMetadataName},
	}

	data := struct{}{}

	fmt.Printf("creating task %s...\n", opts.Name)

	if err := RenderTemplates(renderer, templates, data, false); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	return nil
}
