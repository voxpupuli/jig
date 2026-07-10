// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/voxpupuli/jig/internal/module"
	"github.com/voxpupuli/jig/internal/remote"
)

// templateSource is a resolved template location: a local directory (possibly
// a temporary clone of a remote repository) plus, when remote, the provenance
// to record in metadata.json.
type templateSource struct {
	Dir     string
	URL     string
	Ref     string
	Commit  string
	cleanup func()
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
// template-url recorded in metadataDir's metadata.json, template_dir from the
// config, and finally the embedded templates (empty Dir). metadataDir == ""
// skips the metadata lookup; `new module` uses that since no metadata exists
// yet.
func (a *App) resolveTemplateSource(cmd *cobra.Command, metadataDir string) (*templateSource, error) {
	flags := cmd.InheritedFlags()
	url, _ := flags.GetString("template-url")
	ref, _ := flags.GetString("template-ref")
	dir, _ := flags.GetString("template-dir")

	if url != "" && dir != "" {
		return nil, fmt.Errorf("--template-url and --template-dir are mutually exclusive")
	}

	if url == "" && dir == "" && metadataDir != "" {
		if meta, err := module.ReadMetadata(filepath.Join(metadataDir, "metadata.json")); err == nil {
			url = meta.TemplateURL
			if ref == "" {
				ref = meta.TemplateRef
			}
		}
	}

	if url == "" {
		if ref != "" {
			return nil, fmt.Errorf("--template-ref requires --template-url")
		}
		if dir == "" {
			dir = a.Config.TemplateDir
		}
		return &templateSource{Dir: dir}, nil
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
		cleanup: res.Cleanup,
	}, nil
}
