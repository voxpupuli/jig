// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func NewTransport(opts ComponentOptions) error {
	// Validate the transport name meets the transport name requirements
	// [a-z][a-z0-9_]*
	transportNamePattern := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	if !transportNamePattern.MatchString(opts.Name) {
		return fmt.Errorf(
			"transport name %q does not match the allowed pattern (%s)",
			opts.Name,
			transportNamePattern.String(),
		)
	}

	// Get the metadata, this is to ensure we're in a module directory
	_, err := GetMetadata(opts.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	// Construct file names and check for existing files
	deviceFileName := filepath.Join(opts.WorkDir, "lib", "puppet", "util", "network_device", opts.Name, "device.rb")
	if _, err := os.Stat(deviceFileName); err == nil {
		return fmt.Errorf("device for transport %s already exists: %s", opts.Name, deviceFileName)
	}

	schemaTransportFileName := filepath.Join(opts.WorkDir, "lib", "puppet", "transport", "schema", opts.Name+".rb")
	if _, err := os.Stat(schemaTransportFileName); err == nil {
		return fmt.Errorf("schema transport %s already exists: %s", opts.Name, schemaTransportFileName)
	}

	schemaTransportSpecFileName := filepath.Join(opts.WorkDir, "spec", "unit", "puppet", "transport", "schema", opts.Name+"_spec.rb")
	if _, err := os.Stat(schemaTransportSpecFileName); err == nil {
		return fmt.Errorf("schema transport %s test already exists: %s", opts.Name, schemaTransportSpecFileName)
	}

	transportFileName := filepath.Join(opts.WorkDir, "lib", "puppet", "transport", opts.Name+".rb")
	if _, err := os.Stat(transportFileName); err == nil {
		return fmt.Errorf("transport %s already exists: %s", opts.Name, transportFileName)
	}

	transportSpecFileName := filepath.Join(opts.WorkDir, "spec", "unit", "puppet", "transport", opts.Name+"_spec.rb")
	if _, err := os.Stat(transportSpecFileName); err == nil {
		return fmt.Errorf("transport %s test already exists: %s", opts.Name, transportSpecFileName)
	}

	// Render templates and create the files
	data := struct{ Name string }{Name: opts.Name}
	renderer := newRenderer(opts.TemplateDir)

	templates := []TemplateFile{
		{FileName: "transport/device.rb", Destination: deviceFileName},
		{FileName: "transport/schema_transport.rb", Destination: schemaTransportFileName},
		{FileName: "transport/schema_transport_spec.rb", Destination: schemaTransportSpecFileName},
		{FileName: "transport/transport.rb", Destination: transportFileName},
		{FileName: "transport/transport_spec.rb", Destination: transportSpecFileName},
	}

	fmt.Printf("creating transport %s...\n", opts.Name)

	if err := RenderTemplates(renderer, templates, data, false); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	return nil
}
