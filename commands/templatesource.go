// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/voxpupuli/jig/internal/config"
	"github.com/voxpupuli/jig/internal/module"
	"github.com/voxpupuli/jig/internal/remote"
)

// templateSource is a resolved template location: a local directory (possibly
// a temporary clone of a remote repository) plus, when remote, the provenance
// to record in the module's jig.toml.
type templateSource struct {
	Dir     string
	URL     string
	Ref     string
	Commit  string
	Origin  string // human-readable provenance, e.g. "--template-dir flag"
	cleanup func()
}

// addTemplateSourceFlags registers the persistent flags that feed
// resolveTemplateSource on a parent command, so every subcommand accepts the
// same template source options.
func addTemplateSourceFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("template-dir", "t", "", "Path to custom template directory")
	cmd.PersistentFlags().String("template-url", "", "Git URL of a template repository to clone and use (ssh via ssh-agent, or anonymous http(s))")
	cmd.PersistentFlags().String("template-ref", "", "Git branch, tag, or ref to use with --template-url (default: the remote's default branch)")
	cmd.PersistentFlags().Bool("ssh-accept-new", false, "Automatically trust unknown ssh host keys and add them to known_hosts (changed keys still fail)")
}

// Cleanup removes the temporary clone, if any. Safe to call more than once.
func (s *templateSource) Cleanup() {
	if s.cleanup != nil {
		s.cleanup()
		s.cleanup = nil
	}
}

// resolveTemplateSource decides where templates come from, in order of
// precedence: the --template-url flag, the --template-dir flag, the
// [template] section of moduleDir's jig.toml, template_dir from the config,
// and finally the embedded templates (empty Dir). moduleDir == "" skips the
// module lookup; `new module` uses that since no module exists yet.
// Template keys in metadata.json (written by jig 1.x) are not supported and
// only produce a warning.
func (a *App) resolveTemplateSource(cmd *cobra.Command, moduleDir string) (*templateSource, error) {
	flags := cmd.InheritedFlags()
	url, _ := flags.GetString("template-url")
	ref, _ := flags.GetString("template-ref")
	dir, _ := flags.GetString("template-dir")

	if url != "" && dir != "" {
		return nil, fmt.Errorf("--template-url and --template-dir are mutually exclusive")
	}

	origin := ""
	switch {
	case url != "":
		origin = "--template-url flag"
	case dir != "":
		origin = "--template-dir flag"
	}

	if url == "" && dir == "" && moduleDir != "" {
		moduleConfig, err := config.LoadModuleConfig(moduleDir)
		if err != nil {
			return nil, err
		}
		if moduleConfig.Template.URL != "" {
			url = moduleConfig.Template.URL
			if ref == "" {
				ref = moduleConfig.Template.Ref
			}
			origin = fmt.Sprintf("[template] section of %s", filepath.Join(moduleDir, config.ModuleConfigFileName))
		}
		if meta, err := module.ReadMetadata(filepath.Join(moduleDir, "metadata.json")); err == nil && meta.HasTemplateSettings() {
			fmt.Println("warning: template settings in metadata.json are not supported; move template-url/template-ref/template-commit to the [template] section of jig.toml and remove them from metadata.json")
		}
	}

	if url == "" {
		if ref != "" {
			return nil, fmt.Errorf("--template-ref requires --template-url")
		}
		if dir == "" {
			dir = a.Config.TemplateDir
			if dir != "" {
				origin = "template_dir from config"
			}
		}
		return &templateSource{Dir: dir, Origin: origin}, nil
	}

	acceptNew, _ := flags.GetBool("ssh-accept-new")
	if !acceptNew {
		acceptNew = a.Config.SSHAcceptNew
	}

	if ref != "" {
		fmt.Printf("Fetching templates from %s (ref %s)...\n", url, ref)
	} else {
		fmt.Printf("Fetching templates from %s...\n", url)
	}

	res, err := remote.Fetch(remote.Options{
		URL:          url,
		Ref:          ref,
		SSHAcceptNew: acceptNew,
		In:           os.Stdin,
		Out:          os.Stdout,
	})
	if err != nil {
		return nil, err
	}

	return &templateSource{
		Dir:     res.Dir,
		URL:     url,
		Ref:     ref,
		Commit:  res.Commit,
		Origin:  origin,
		cleanup: res.Cleanup,
	}, nil
}
