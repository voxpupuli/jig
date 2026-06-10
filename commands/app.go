// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"github.com/avitacco/jig/internal/config"
	"github.com/sirupsen/logrus"
)

type App struct {
	Config config.Config
	Logger *logrus.Logger
}

func NewApp() *App {
	return &App{
		Logger: logrus.New(),
	}
}
