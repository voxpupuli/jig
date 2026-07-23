// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version is stamped by release builds via
//
//	-ldflags "-X github.com/voxpupuli/jig/commands.version=x.y.z"
//
// and stays empty for plain `go build` / `go install` builds, where Version
// falls back to the module version Go embeds in the binary.
var version = ""

// Version reports the best available version string: the release-stamped
// value; else the commit hash Go embeds when building from a git checkout;
// else the module version (set for `go install module@version`); else "dev".
func Version() string {
	if version != "" {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	var revision string
	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if revision != "" {
		if len(revision) > 12 {
			revision = revision[:12]
		}
		if dirty {
			revision += "+dirty"
		}
		return "dev (commit " + revision + ")"
	}

	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func (a *App) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the jig version",
		// Override the root's config-loading hook: version must work even
		// when the config file is broken (e.g. while reporting a bug).
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "jig %s\n", Version())
		},
	}
}
