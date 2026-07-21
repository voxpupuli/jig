// SPDX-License-Identifier: GPL-3.0-or-later
package module

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Metadata struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Author          string            `json:"author"`
	License         string            `json:"license"`
	Summary         string            `json:"summary"`
	Source          string            `json:"source"`
	ProjectPage     string            `json:"project_page,omitempty"`
	IssuesURL       string            `json:"issues_url,omitempty"`
	Dependencies    []Dependency      `json:"dependencies"`
	Requirements    []Requirement     `json:"requirements"`
	OperatingSystem []OperatingSystem `json:"operatingsystem_support"`
	Tags            []string          `json:"tags"`
	PdkVersion      string            `json:"pdk-version"`
	// TemplateURL, TemplateRef, and TemplateCommit were written by jig 1.x
	// to record which template repository the module was scaffolded from.
	// jig 2.x records template provenance in jig.toml instead and does not
	// support these keys; they are parsed only so their presence can be
	// detected and warned about. Trust-related settings (like
	// ssh-accept-new) deliberately never live in either file: both are
	// shared via the module repository and must not be able to change
	// security decisions for other users.
	TemplateURL    string `json:"template-url,omitempty"`
	TemplateRef    string `json:"template-ref,omitempty"`
	TemplateCommit string `json:"template-commit,omitempty"`
}

type Dependency struct {
	Name               string `json:"name"`
	VersionRequirement string `json:"version_requirement"`
}

type Requirement struct {
	Name               string `json:"name"`
	VersionRequirement string `json:"version_requirement"`
}

type OperatingSystem struct {
	Name    string   `json:"operatingsystem"`
	Release []string `json:"operatingsystemrelease"`
}

func NewMetadata(name string, forgeUser string, author string) Metadata {
	return Metadata{
		Name:         fmt.Sprintf("%s-%s", forgeUser, name),
		Version:      "0.1.0",
		Author:       author,
		License:      "Apache-2.0",
		Summary:      "",
		Source:       "",
		ProjectPage:  "",
		IssuesURL:    "",
		Dependencies: []Dependency{},
		Requirements: []Requirement{
			{
				Name:               "openvox",
				VersionRequirement: ">= 7.0.0 < 9.0.0",
			},
		},
		OperatingSystem: []OperatingSystem{},
		Tags:            []string{},
		PdkVersion:      "3.4.0",
	}
}

func ReadMetadata(path string) (Metadata, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Metadata{}, err
	}

	metadata := Metadata{}
	err = json.Unmarshal(content, &metadata)
	if err != nil {
		return Metadata{}, err
	}
	return metadata, nil
}

func (m Metadata) Write(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Using json.Encoder here instead of json.MarshalIndent to avoid escaping HTML
	encoder := json.NewEncoder(f)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(m)
}

// HasTemplateSettings reports whether the metadata carries any of the
// unsupported jig 1.x template keys, so callers can warn about them.
func (m Metadata) HasTemplateSettings() bool {
	return m.TemplateURL != "" || m.TemplateRef != "" || m.TemplateCommit != ""
}

func (m Metadata) ModuleName() string {
	parts := strings.SplitN(m.Name, "-", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return m.Name
}

func (m Metadata) ForgeUsername() string {
	parts := strings.SplitN(m.Name, "-", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}
