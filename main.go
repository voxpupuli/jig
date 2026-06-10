// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"os"

	"github.com/avitacco/jig/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
