// SPDX-License-Identifier: GPL-3.0-or-later
package template

import (
	"fmt"
	"maps"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Variable types a template may declare in jig-template.toml.
const (
	VarTypeBool   = "bool"
	VarTypeString = "string"
	VarTypeInt    = "int"
	VarTypeList   = "list"
)

// varNamePattern is the only accepted shape for a variable name. Lowercase
// is required because the user config goes through viper, which folds keys
// to lowercase: a camelCase name would silently stop matching the
// template's .Vars reference.
var varNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ValidateVarName rejects names that are not lowercase snake_case.
func ValidateVarName(name string) error {
	if !varNamePattern.MatchString(name) {
		return fmt.Errorf("invalid template variable name %q: must match %s", name, varNamePattern)
	}
	return nil
}

// ValidateVarValue checks that a stored value is a plain value: a string,
// boolean, number, or a list of those. Tables are rejected because in the
// user config a table under an author is how a module section is written,
// so a table-valued variable could not be told apart from one.
func ValidateVarValue(name string, value any) error {
	switch v := value.(type) {
	case string, bool, int, int64, float64:
		return nil
	case []string:
		return nil
	case []any:
		for _, elem := range v {
			switch elem.(type) {
			case string, bool, int, int64, float64:
			default:
				return fmt.Errorf("template variable %q: list elements must be plain values, got %T", name, elem)
			}
		}
		return nil
	default:
		return fmt.Errorf("template variable %q: value must be a string, boolean, number, or list, got %T", name, value)
	}
}

// ValidateVars validates every name and value in vars.
func ValidateVars(vars map[string]any) error {
	for _, name := range sortedKeys(vars) {
		if err := ValidateVarName(name); err != nil {
			return err
		}
		if err := ValidateVarValue(name, vars[name]); err != nil {
			return err
		}
	}
	return nil
}

// ParseVarFlags parses repeated --template-var name=value flags. Repeating a
// name collects its values in order, which is how a list variable is given
// on the command line.
func ParseVarFlags(flags []string) (map[string][]string, error) {
	parsed := map[string][]string{}
	for _, flag := range flags {
		name, value, ok := strings.Cut(flag, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --template-var %q: expected name=value", flag)
		}
		if err := ValidateVarName(name); err != nil {
			return nil, err
		}
		parsed[name] = append(parsed[name], value)
	}
	return parsed, nil
}

// coerceString converts one command-line or interview value to the declared
// type of spec. A list value is taken as a single element.
func (spec VarSpec) coerceString(name string, raw []string) (any, error) {
	if spec.Type == VarTypeList {
		return append([]string{}, raw...), nil
	}
	if len(raw) != 1 {
		return nil, fmt.Errorf("template variable %q is a %s and takes one value, got %d", name, spec.Type, len(raw))
	}
	switch spec.Type {
	case VarTypeBool:
		b, err := strconv.ParseBool(raw[0])
		if err != nil {
			return nil, fmt.Errorf("template variable %q is a bool, got %q", name, raw[0])
		}
		return b, nil
	case VarTypeInt:
		n, err := strconv.ParseInt(raw[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("template variable %q is an int, got %q", name, raw[0])
		}
		return n, nil
	default:
		return raw[0], nil
	}
}

// normalize checks a stored value (from jig.toml, the user config, or a
// declared default) against the declared type and returns it in the form
// templates see: int64 for ints and []string for lists.
func (spec VarSpec) normalize(name string, value any) (any, error) {
	switch spec.Type {
	case VarTypeBool:
		if b, ok := value.(bool); ok {
			return b, nil
		}
	case VarTypeString:
		if s, ok := value.(string); ok {
			return s, nil
		}
	case VarTypeInt:
		switch n := value.(type) {
		case int64:
			return n, nil
		case int:
			return int64(n), nil
		}
	case VarTypeList:
		switch l := value.(type) {
		case []string:
			return append([]string{}, l...), nil
		case []any:
			out := make([]string, 0, len(l))
			for _, elem := range l {
				s, ok := elem.(string)
				if !ok {
					return nil, fmt.Errorf("template variable %q is a list of strings, got element %v (%T)", name, elem, elem)
				}
				out = append(out, s)
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("template variable %q is a %s, got %v (%T)", name, spec.Type, value, value)
}

// zero returns the value a declared variable takes when it has no default.
func (spec VarSpec) zero() any {
	switch spec.Type {
	case VarTypeBool:
		return false
	case VarTypeInt:
		return int64(0)
	case VarTypeList:
		return []string{}
	default:
		return ""
	}
}

// FormatVarValue renders a resolved value the way it would be typed back in,
// for interview prompts.
func FormatVarValue(value any) string {
	switch v := value.(type) {
	case []string:
		return strings.Join(v, ", ")
	default:
		return fmt.Sprint(v)
	}
}

// ParseVarInput converts one interview answer for a declared variable. Lists
// are given comma-separated.
func (m Manifest) ParseVarInput(name, input string) (any, error) {
	spec, ok := m.Vars[name]
	if !ok {
		return nil, fmt.Errorf("the template does not declare a variable named %q", name)
	}
	raw := []string{input}
	if spec.Type == VarTypeList {
		raw = nil
		for part := range strings.SplitSeq(input, ",") {
			if part = strings.TrimSpace(part); part != "" {
				raw = append(raw, part)
			}
		}
	}
	return spec.coerceString(name, raw)
}

// CoerceFlags converts parsed --template-var values to the declared types.
// A name the template does not declare is an error: it is almost always a
// typo, and silently ignoring it would render the module without it.
func (m Manifest) CoerceFlags(flags map[string][]string) (map[string]any, error) {
	out := make(map[string]any, len(flags))
	for _, name := range sortedKeys(flags) {
		spec, ok := m.Vars[name]
		if !ok {
			return nil, fmt.Errorf("--template-var %s: the template does not declare a variable named %q", name, name)
		}
		value, err := spec.coerceString(name, flags[name])
		if err != nil {
			return nil, err
		}
		out[name] = value
	}
	return out, nil
}

// VarLayers are the sources a variable's value can come from, apart from
// command-line flags and the template's declared default.
type VarLayers struct {
	// Module holds the module's stored values ([template.vars] in jig.toml).
	// Names the template does not declare are passed through unchanged, so
	// a variable a template dropped is not lost from the module.
	Module map[string]any
	// Global holds user config layers, most specific first. Names the
	// template does not declare are ignored: the user config is shared by
	// every template the user scaffolds from.
	Global []map[string]any
}

// ResolveVars computes the variables a render sees, with precedence: flags,
// then layers.Module, then each of layers.Global in order, then the
// template's declared default. Every declared variable is present in the
// result.
func (m Manifest) ResolveVars(flags map[string]any, layers VarLayers) (map[string]any, error) {
	resolved := map[string]any{}

	for _, name := range sortedKeys(layers.Module) {
		value := layers.Module[name]
		if err := ValidateVarValue(name, value); err != nil {
			return nil, err
		}
		spec, declared := m.Vars[name]
		if !declared {
			resolved[name] = value
			continue
		}
		normalized, err := spec.normalize(name, value)
		if err != nil {
			return nil, err
		}
		resolved[name] = normalized
	}

	for name, spec := range m.Vars {
		if _, ok := resolved[name]; ok {
			continue
		}
		value := spec.Default
		for _, layer := range layers.Global {
			if v, ok := layer[name]; ok {
				normalized, err := spec.normalize(name, v)
				if err != nil {
					return nil, fmt.Errorf("user config: %w", err)
				}
				value = normalized
				break
			}
		}
		resolved[name] = value
	}

	maps.Copy(resolved, flags)
	return resolved, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
