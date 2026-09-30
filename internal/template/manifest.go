// SPDX-License-Identifier: GPL-3.0-or-later
package template

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	tmplpkg "text/template"

	gogitignore "github.com/go-git/go-git/v5/plumbing/format/gitignore"
	toml "github.com/pelletier/go-toml/v2"
)

// ManifestFileName is the file at the root of a template directory that
// declares the template's variables and conditional files.
const ManifestFileName = "jig-template.toml"

// VarSpec declares one template variable.
type VarSpec struct {
	Type        string `toml:"type"`
	Default     any    `toml:"default"`
	Description string `toml:"description"`
	// Prompt makes the jig new module interview ask for the variable.
	Prompt bool `toml:"prompt"`
}

// FileRule makes the files matching Path (a gitignore-style glob relative to
// the generated module) depend on When, a text/template expression that
// must evaluate to true or false.
type FileRule struct {
	Path string `toml:"path"`
	When string `toml:"when"`

	pattern gogitignore.Pattern
	when    *tmplpkg.Template
}

// Manifest is a template directory's jig-template.toml. The zero Manifest
// (no file, or the embedded templates) declares nothing.
type Manifest struct {
	Vars  map[string]VarSpec `toml:"vars"`
	Files []FileRule         `toml:"files"`
}

// LoadManifest reads jig-template.toml from the root of a template
// directory. An empty dir (the embedded templates) or a missing file yields
// the zero Manifest. Everything is validated here, including parsing every
// when expression, so a broken manifest fails before any file is written.
func LoadManifest(dir string) (Manifest, error) {
	if dir == "" {
		return Manifest{}, nil
	}
	path := filepath.Join(dir, ManifestFileName)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Manifest{}, nil
		}
		return Manifest{}, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var m Manifest
	dec := toml.NewDecoder(bytes.NewReader(content))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		var strict *toml.StrictMissingError
		if errors.As(err, &strict) {
			return Manifest{}, fmt.Errorf("unknown keys in %s:\n%s", path, strict.String())
		}
		return Manifest{}, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	if err := m.validate(); err != nil {
		return Manifest{}, fmt.Errorf("invalid %s: %w", path, err)
	}
	return m, nil
}

func (m *Manifest) validate() error {
	for _, name := range sortedKeys(m.Vars) {
		spec := m.Vars[name]
		if err := ValidateVarName(name); err != nil {
			return err
		}
		switch spec.Type {
		case VarTypeBool, VarTypeString, VarTypeInt, VarTypeList:
		case "":
			return fmt.Errorf("variable %q: type is required (%s, %s, %s, or %s)", name, VarTypeBool, VarTypeString, VarTypeInt, VarTypeList)
		default:
			return fmt.Errorf("variable %q: unknown type %q (want %s, %s, %s, or %s)", name, spec.Type, VarTypeBool, VarTypeString, VarTypeInt, VarTypeList)
		}
		if spec.Default == nil {
			spec.Default = spec.zero()
		} else {
			normalized, err := spec.normalize(name, spec.Default)
			if err != nil {
				return fmt.Errorf("default: %w", err)
			}
			spec.Default = normalized
		}
		m.Vars[name] = spec
	}

	for i := range m.Files {
		rule := &m.Files[i]
		if rule.Path == "" {
			return fmt.Errorf("files[%d]: path is required", i)
		}
		if strings.HasPrefix(rule.Path, "!") {
			return fmt.Errorf("files[%d]: negated path %q is not supported", i, rule.Path)
		}
		if strings.TrimSpace(rule.When) == "" {
			return fmt.Errorf("files[%d] (%s): when is required", i, rule.Path)
		}
		rule.pattern = gogitignore.ParsePattern(rule.Path, nil)

		t, err := tmplpkg.New(rule.Path).Funcs(funcMap).Parse("{{ " + rule.When + " }}")
		if err != nil {
			return fmt.Errorf("files[%d] (%s): invalid when expression %q: %w", i, rule.Path, rule.When, err)
		}
		for _, name := range varRefs(t) {
			if _, ok := m.Vars[name]; !ok {
				return fmt.Errorf("files[%d] (%s): when expression uses .Vars.%s, which is not declared", i, rule.Path, name)
			}
		}
		rule.when = t
	}
	return nil
}

// Declares reports whether the template declares a variable named name.
func (m Manifest) Declares(name string) bool {
	_, ok := m.Vars[name]
	return ok
}

// Includes reports whether the module file at path (slash-separated,
// relative to the module root) is generated with the given data. A file is
// generated unless a [[files]] rule matching it has a when expression that
// evaluates to false. A when expression that evaluates to anything other
// than true or false is an error.
func (m Manifest) Includes(path string, data any) (bool, error) {
	parts := strings.Split(path, "/")
	for _, rule := range m.Files {
		if rule.pattern.Match(parts, false) != gogitignore.Exclude {
			continue
		}
		var buf bytes.Buffer
		if err := rule.when.Execute(&buf, data); err != nil {
			return false, fmt.Errorf("evaluating when %q for %s: %w", rule.When, path, err)
		}
		result := strings.TrimSpace(buf.String())
		include, err := strconv.ParseBool(result)
		if err != nil {
			return false, fmt.Errorf("when %q for %s must evaluate to true or false, got %q", rule.When, path, result)
		}
		if !include {
			return false, nil
		}
	}
	return true, nil
}
