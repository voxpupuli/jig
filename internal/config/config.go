// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	ForgeUsername string `mapstructure:"forge_username"`
	Author        string `mapstructure:"author"`
	License       string `mapstructure:"license"`
	ForgeToken    string `mapstructure:"forge_token"`
	TemplateDir   string `mapstructure:"template_dir"`
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

	viper.SetConfigFile(path)
	viper.SetEnvPrefix("JIG")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return Config{}, err
		}
		logger.Debugf("no config file found at %s, using default values", path)
	} else {
		logger.Debugf("loaded config from %s", path)
	}

	config := Config{}
	err := viper.Unmarshal(&config)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}
