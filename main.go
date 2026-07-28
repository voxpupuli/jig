// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"os"

	"github.com/voxpupuli/jig/v2/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
