// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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
	Runner        RunnerConfig `mapstructure:"runner"`
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
	return config, nil
}
