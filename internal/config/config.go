// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/voxpupuli/jig/v2/internal/template"
)

// Default values for the container runner. Exposed so callers and tests can
// refer to them without duplicating the literals.
const (
	DefaultRunnerType   = "local"
	DefaultRunnerEngine = "docker"
	DefaultRunnerImage  = "ghcr.io/voxpupuli/voxbox:latest"
)

type Config struct {
	ForgeUsername string       `mapstructure:"forge_username"`
	Author        string       `mapstructure:"author"`
	License       string       `mapstructure:"license"`
	ForgeToken    string       `mapstructure:"forge_token"`
	TemplateDir   string       `mapstructure:"template_dir"`
	SSHAcceptNew  bool         `mapstructure:"ssh_accept_new"`
	Runner        RunnerConfig `mapstructure:"runner"`
	// TemplateVars is the raw [template.vars] table: defaults for template
	// variables, applied only when jig new module or jig convert first
	// records a module's values. From a TOML config it is read straight
	// from the file rather than through viper, which lowercases keys and
	// would hide a non-snake_case name instead of rejecting it.
	TemplateVars map[string]any `mapstructure:"-"`
}

// RunnerConfig controls how the bundle-backed commands (update/test/validate)
// are executed. With Type "local" jig invokes the host's `bundle` directly;
// with "voxbox" it runs `bundle` inside the voxbox container so no system-wide
// Ruby/bundler install is required (e.g. on Windows).
type RunnerConfig struct {
	Type   string `mapstructure:"type"`
	Engine string `mapstructure:"engine"`
	Image  string `mapstructure:"image"`
}

func Load(path string, logger *logrus.Logger) (Config, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, err
		}
		path = filepath.Join(home, ".config", "jig", "config.toml")
		logger.Debugf("config path not provided, using default path: %s", path)
	}

	// A fresh instance keeps loads isolated from each other (and from any
	// global viper state), which matters for both repeated calls and tests.
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix("JIG")
	// Map nested keys onto env vars, e.g. runner.type -> JIG_RUNNER_TYPE.
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Defaults double as key registration so AutomaticEnv overrides are picked
	// up by Unmarshal even when the keys are absent from the config file.
	v.SetDefault("ssh_accept_new", false)
	v.SetDefault("runner.type", DefaultRunnerType)
	v.SetDefault("runner.engine", DefaultRunnerEngine)
	v.SetDefault("runner.image", DefaultRunnerImage)

	if err := v.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return Config{}, err
		}
		logger.Debugf("no config file found at %s, using default values", path)
	} else {
		logger.Debugf("loaded config from %s", path)
	}

	config := Config{}
	err := v.Unmarshal(&config)
	if err != nil {
		return Config{}, err
	}

	if isTOML(path) {
		vars, err := readTemplateVars(path)
		if err != nil {
			return Config{}, err
		}
		config.TemplateVars = vars
	} else if vars, ok := v.Get("template.vars").(map[string]any); ok {
		// Other formats viper supports (--config foo.yaml) keep working;
		// their keys arrive lowercased, so case mistakes go undetected.
		config.TemplateVars = vars
	}
	return config, nil
}

func isTOML(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".toml" || ext == ""
}

// readTemplateVars reads the [template.vars] table from a TOML config file,
// preserving key case. A missing file has none.
func readTemplateVars(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw struct {
		Template struct {
			Vars map[string]any `toml:"vars"`
		} `toml:"template"`
	}
	if err := toml.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return raw.Template.Vars, nil
}

// globalVarsDefault is the [template.vars] section applying to every module.
// It is never read as an author name.
const globalVarsDefault = "default"

// GlobalTemplateVars returns the user config layers for a module, most
// specific first: [template.vars.<author>.<module>],
// [template.vars.<author>], then [template.vars.default]. The whole
// [template.vars] table is validated, not just the sections that apply, so
// a mistake anywhere in it is reported.
func (c Config) GlobalTemplateVars(author, module string) ([]map[string]any, error) {
	var defaults, authorVars, moduleVars map[string]any

	for _, key := range sortedKeys(c.TemplateVars) {
		section, ok := c.TemplateVars[key].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("config: template.vars.%s: put defaults for every module under [template.vars.default]", key)
		}

		if key == globalVarsDefault {
			if err := validateVarSection("template.vars.default", section); err != nil {
				return nil, err
			}
			defaults = section
			continue
		}

		vars := map[string]any{}
		for _, name := range sortedKeys(section) {
			value := section[name]
			if modSection, isTable := value.(map[string]any); isTable {
				if err := validateVarSection("template.vars."+key+"."+name, modSection); err != nil {
					return nil, err
				}
				if key == author && name == module {
					moduleVars = modSection
				}
				continue
			}
			if err := template.ValidateVarName(name); err != nil {
				return nil, fmt.Errorf("config: template.vars.%s: %w", key, err)
			}
			if err := template.ValidateVarValue(name, value); err != nil {
				return nil, fmt.Errorf("config: template.vars.%s: %w", key, err)
			}
			vars[name] = value
		}
		if key == author {
			authorVars = vars
		}
	}

	var layers []map[string]any
	for _, layer := range []map[string]any{moduleVars, authorVars, defaults} {
		if len(layer) > 0 {
			layers = append(layers, layer)
		}
	}
	return layers, nil
}

// ValidateTemplateVars checks the whole [template.vars] table, so commands
// can report a mistake in it before asking any interview questions.
func (c Config) ValidateTemplateVars() error {
	_, err := c.GlobalTemplateVars("", "")
	return err
}

// validateVarSection checks a section holding only variables: tables are
// not allowed in it.
func validateVarSection(where string, section map[string]any) error {
	for _, name := range sortedKeys(section) {
		if _, isTable := section[name].(map[string]any); isTable {
			return fmt.Errorf("config: %s.%s: variable values must be plain values or lists, not tables", where, name)
		}
	}
	if err := template.ValidateVars(section); err != nil {
		return fmt.Errorf("config: %s: %w", where, err)
	}
	return nil
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
