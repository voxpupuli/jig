// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"github.com/avitacco/jig/internal/bundle"
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

// runner builds the bundle.Runner from the loaded config so the
// update/test/validate commands can transparently run locally or via voxbox.
func (a *App) runner() bundle.Runner {
	return bundle.Runner{
		Type:   a.Config.Runner.Type,
		Engine: a.Config.Runner.Engine,
		Image:  a.Config.Runner.Image,
	}
}

// runRake runs a rake task (e.g. spec, validate lint) honouring the runner.
func (a *App) runRake(args []string) error {
	return bundle.RunRake(a.runner(), args)
}

// runBundle runs a raw bundle command (e.g. exec msync update) honouring the
// runner.
func (a *App) runBundle(args []string) error {
	return bundle.RunBundle(a.runner(), args)
}
