// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/voxpupuli/jig/v2/internal/template"
)

// ModuleConfigFileName is the per-module config file, located in the module
// root next to metadata.json.
const ModuleConfigFileName = "jig.toml"

// Build actions. The action is the default treatment for every file when
// building the module package; the exceptions list holds the paths treated
// the opposite way.
const (
	// BuildActionAllow packages everything except the exceptions (a
	// denylist workflow, replacing the old .pdkignore behavior).
	BuildActionAllow = "allow"
	// BuildActionDeny packages nothing except the exceptions. With no
	// exceptions configured this is the spec allowlist, which is also the
	// behavior when jig.toml or its [build] section is absent. User
	// exceptions extend the built-in spec allowlist.
	BuildActionDeny = "deny"
)

// ModuleConfig is the per-module jig.toml. Unlike the global config it is
// committed to the module repository and shared by everyone working on the
// module. Trust-related settings (like ssh_accept_new) must never live here:
// a cloned repository must not be able to change security decisions for
// other users.
type ModuleConfig struct {
	Template ModuleTemplate `toml:"template,omitempty"`
	Renew    RenewConfig    `toml:"renew,omitempty"`
	Build    BuildConfig    `toml:"build,omitempty"`
}

// ModuleTemplate records which template source the module was scaffolded
// from, so later jig invocations in the module default to the same source.
// jig 1.x recorded these in metadata.json (template-url etc.); jig 2.x
// records them here.
type ModuleTemplate struct {
	URL    string `toml:"url,omitempty"`
	Ref    string `toml:"ref,omitempty"`
	Commit string `toml:"commit,omitempty"`
	// Vars holds the module's values for the variables its templates
	// declare. It is the source of truth for renew and for scaffolding
	// components inside the module.
	Vars map[string]any `toml:"vars,omitempty"`
}

// VarsEqual reports whether a and b would be written to jig.toml the same
// way. Comparing the serialized form matters because a list read back from
// jig.toml is a []any while the same list after resolution is a []string.
func VarsEqual(a, b map[string]any) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	ea, errA := toml.Marshal(a)
	eb, errB := toml.Marshal(b)
	return errA == nil && errB == nil && string(ea) == string(eb)
}

// SameSource reports whether t and o record the same template source,
// ignoring Vars.
func (t ModuleTemplate) SameSource(o ModuleTemplate) bool {
	return t.URL == o.URL && t.Ref == o.Ref && t.Commit == o.Commit
}

// RenewConfig controls `jig renew`. Paths is the allowlist of files (globs)
// that renew may re-render and overwrite. It is empty by default so that
// nothing can be overwritten accidentally.
type RenewConfig struct {
	Paths []string `toml:"paths,omitempty"`
}

// BuildConfig controls which files go into the module package. Action is the
// default treatment for every file ("deny" when empty); Exceptions are
// gitignore-style globs treated the opposite way. See the BuildAction
// constants for the exact semantics.
type BuildConfig struct {
	Action     string   `toml:"action,omitempty"`
	Exceptions []string `toml:"exceptions,omitempty"`
}

// Validate checks the build section for values the build cannot interpret.
func (b BuildConfig) Validate() error {
	switch b.Action {
	case "", BuildActionAllow, BuildActionDeny:
		return nil
	default:
		return fmt.Errorf("invalid build action %q: must be %q or %q", b.Action, BuildActionAllow, BuildActionDeny)
	}
}

// LoadModuleConfig reads jig.toml from the module directory. A missing file
// is not an error: it returns the zero config, which yields the default
// behavior everywhere (spec-allowlist builds, empty renew allowlist, no
// recorded template source).
func LoadModuleConfig(dir string) (ModuleConfig, error) {
	path := filepath.Join(dir, ModuleConfigFileName)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ModuleConfig{}, nil
		}
		return ModuleConfig{}, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var cfg ModuleConfig
	if err := toml.Unmarshal(content, &cfg); err != nil {
		return ModuleConfig{}, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	if err := cfg.Build.Validate(); err != nil {
		return ModuleConfig{}, fmt.Errorf("invalid %s: %w", path, err)
	}
	if err := template.ValidateVars(cfg.Template.Vars); err != nil {
		return ModuleConfig{}, fmt.Errorf("invalid [template.vars] in %s: %w", path, err)
	}
	return cfg, nil
}

// moduleConfigHeader introduces the generated jig.toml. Sections are
// documented here rather than emitted with placeholder values so that an
// absent section keeps meaning "jig's defaults".
const moduleConfigHeader = `# Per-module jig configuration.
#
# [template]  url/ref/commit of the template repository this module was
#             scaffolded from; later jig commands in this module default to it.
# [template.vars]
#             values for the variables the templates declare; jig renew and
#             jig new <component> render with these.
# [renew]     paths = [...] -- files jig renew may re-render and overwrite.
#             Empty by default so nothing is overwritten accidentally.
# [build]     action = "allow" or "deny" (default "deny") with exceptions =
#             [...]. Deny packages only the Puppet module spec allowlist plus
#             the exceptions; allow packages everything except the exceptions.
`

// Write saves the config as jig.toml in the module directory.
func (c ModuleConfig) Write(dir string) error {
	content, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to serialize module config: %w", err)
	}
	path := filepath.Join(dir, ModuleConfigFileName)
	if err := os.WriteFile(path, append([]byte(moduleConfigHeader+"\n"), content...), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}
