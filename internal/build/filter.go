// SPDX-License-Identifier: GPL-3.0-or-later
package build

import (
	"os"
	"path/filepath"
	"strings"

	gogitignore "github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/voxpupuli/jig/internal/config"
)

// specAllowlist is the set of files permitted in a published module per the
// Puppet module specification. It mirrors the allowlist puppet-modulebuilder
// uses (lib/puppet/modulebuilder/builder.rb), which Vox Pupuli builds with
// today. It is the implicit exception list when a module's [build] section
// uses action "deny" (the default); configured exceptions extend it.
var specAllowlist = []string{
	"/CHANGELOG*",
	"/LICENSE",
	"/README*",
	"/REFERENCE.md",
	"/bolt_plugin.json",
	"/data/**",
	"/docs/**",
	"/examples/**",
	"/facts.d/**",
	"/files/**",
	"/functions/**",
	"/hiera.yaml",
	"/lib/**",
	"/locales/**",
	"/manifests/**",
	"/metadata.json",
	"/plans/**",
	"/scripts/**",
	"/tasks/**",
	"/templates/**",
	"/types/**",
}

// alwaysExcluded are paths that never belong in a package regardless of the
// configured action: jig's own state (pkg output, module config), version
// control internals, and empty-directory markers.
var alwaysExcluded = []string{
	"/pkg",
	"/.git",
	"/" + config.ModuleConfigFileName,
	".gitkeep",
}

// findIgnoreFiles returns ignore-style files (.pdkignore, .pmtignore, and any
// other dotfile ending in "ignore") present in the module root. jig packaging
// is controlled solely by the [build] section of jig.toml, so a leftover
// ignore file means the module expects behavior jig does not have -- the
// build warns about each one. .gitignore is exempt: it belongs to git, has a
// purpose beyond packaging, and is scaffolded by jig itself.
func findIgnoreFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var found []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == ".gitignore" {
			continue
		}
		if strings.HasPrefix(name, ".") && strings.HasSuffix(name, "ignore") {
			found = append(found, filepath.Join(dir, name))
		}
	}
	return found
}

// buildFilter decides which files go into the module archive, according to
// the [build] section of jig.toml: the action is the default treatment for
// every file, the exceptions are treated the opposite way.
type buildFilter struct {
	action     string
	exceptions gogitignore.Matcher
	excluded   gogitignore.Matcher
}

func newBuildFilter(cfg config.BuildConfig) (*buildFilter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	action := cfg.Action
	if action == "" {
		action = config.BuildActionDeny
	}

	var patterns []gogitignore.Pattern
	if action == config.BuildActionDeny {
		for _, p := range specAllowlist {
			patterns = append(patterns, gogitignore.ParsePattern(p, nil))
		}
	}
	for _, p := range cfg.Exceptions {
		patterns = append(patterns, gogitignore.ParsePattern(p, nil))
	}

	var excluded []gogitignore.Pattern
	for _, p := range alwaysExcluded {
		excluded = append(excluded, gogitignore.ParsePattern(p, nil))
	}

	return &buildFilter{
		action:     action,
		exceptions: gogitignore.NewMatcher(patterns),
		excluded:   gogitignore.NewMatcher(excluded),
	}, nil
}

// Include reports whether the file at relPath (slash-separated, relative to
// the module root) belongs in the archive.
func (f *buildFilter) Include(relPath string) bool {
	parts := strings.Split(relPath, "/")
	if f.excluded.Match(parts, false) {
		return false
	}
	matched := f.exceptions.Match(parts, false)
	if f.action == config.BuildActionDeny {
		return matched
	}
	return !matched
}
