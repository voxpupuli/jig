// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"os"
	"regexp"
	"strings"

	"github.com/voxpupuli/jig/v2/internal/module"
)

// ModulefileData is the subset of a Puppet 3.x-era Modulefile that maps onto
// metadata.json. Modulefile is a small Ruby DSL (`name 'x'`, `dependency
// 'a/b', '>= 1.0.0'`, ...); a line-based parse of the keys below covers real-
// world files without needing a Ruby interpreter, matching what `puppet
// module build` did with the same files.
type ModulefileData struct {
	Name         string
	Version      string
	Author       string
	License      string
	Summary      string
	Description  string
	Source       string
	ProjectPage  string
	Dependencies []module.Dependency
}

var (
	modulefileLineRe = regexp.MustCompile(`^([a-zA-Z_]+)\s+(.+)$`)
	quotedRe         = regexp.MustCompile(`'([^']*)'|"([^"]*)"`)
)

// extractQuoted returns every single- or double-quoted substring in s, in
// order.
func extractQuoted(s string) []string {
	matches := quotedRe.FindAllStringSubmatch(s, -1)
	values := make([]string, 0, len(matches))
	for _, m := range matches {
		if m[1] != "" {
			values = append(values, m[1])
		} else {
			values = append(values, m[2])
		}
	}
	return values
}

// ParseModulefile reads a Modulefile and extracts the keys jig knows how to
// carry forward into metadata.json. Keys it doesn't recognize are ignored.
func ParseModulefile(path string) (ModulefileData, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ModulefileData{}, err
	}

	var data ModulefileData
	for _, line := range joinContinuations(strings.Split(string(content), "\n")) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		m := modulefileLineRe.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}

		key := strings.ToLower(m[1])
		values := extractQuoted(m[2])
		if len(values) == 0 {
			continue
		}

		switch key {
		case "name":
			data.Name = values[0]
		case "version":
			data.Version = values[0]
		case "author":
			data.Author = values[0]
		case "license":
			data.License = values[0]
		case "summary":
			data.Summary = values[0]
		case "description":
			data.Description = values[0]
		case "source":
			data.Source = values[0]
		case "project_page":
			data.ProjectPage = values[0]
		case "dependency":
			dep := module.Dependency{Name: normalizeDependencyName(values[0])}
			if len(values) > 1 {
				dep.VersionRequirement = values[1]
			}
			data.Dependencies = append(data.Dependencies, dep)
		}
	}

	return data, nil
}

// joinContinuations merges a Ruby-style multi-line call, such as
//
//	dependency 'puppetlabs/stdlib',
//	           '>= 4.0.0'
//
// into a single logical line, so the line-based parser above sees it as one
// statement. A trailing comma is the continuation signal; lines are joined
// with a space until one is found that doesn't end with one.
func joinContinuations(lines []string) []string {
	var result []string
	pending := ""
	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t\r")
		if pending != "" {
			pending += " " + strings.TrimSpace(line)
		} else {
			pending = line
		}
		if strings.HasSuffix(strings.TrimSpace(pending), ",") {
			continue
		}
		result = append(result, pending)
		pending = ""
	}
	if pending != "" {
		result = append(result, pending)
	}
	return result
}

// normalizeDependencyName rewrites the legacy "forgeuser/modulename" form
// Modulefile accepted to the "forgeuser-modulename" form current tooling
// (and linters) expect. Anything else -- no slash, or more than one -- is
// left alone.
func normalizeDependencyName(name string) string {
	if strings.Count(name, "/") != 1 {
		return name
	}
	return strings.Replace(name, "/", "-", 1)
}

// SplitForgeName splits a Forge-convention "forgeuser-modulename" or
// "forgeuser/modulename" string (both accepted by Puppet's own Modulefile
// and metadata.json name validation) into its two parts. A string with
// neither separator is returned as the module name with an empty forge
// user.
func SplitForgeName(raw string) (forgeUser string, name string) {
	idx := strings.IndexAny(raw, "-/")
	if idx == -1 {
		return "", raw
	}
	return raw[:idx], raw[idx+1:]
}
