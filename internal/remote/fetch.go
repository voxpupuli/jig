// SPDX-License-Identifier: GPL-3.0-or-later

// Package remote fetches template repositories from git URLs into a
// temporary directory so they can be used exactly like a local template
// directory. Only ssh (via a running ssh-agent) and anonymous http(s)
// transports are supported.
package remote

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// Options controls how a remote template repository is fetched.
type Options struct {
	// URL is the git URL of the template repository (ssh, http(s), or a
	// local path).
	URL string
	// Ref is the branch, tag, or fully qualified ref name to check out.
	// Empty means the remote's default branch.
	Ref string
	// SSHAcceptNew automatically trusts unknown ssh host keys, like
	// OpenSSH's StrictHostKeyChecking=accept-new. A host whose recorded
	// key has changed still fails hard.
	SSHAcceptNew bool
	// In is where interactive prompts read from (typically os.Stdin).
	// Prompting only happens when In is a terminal.
	In io.Reader
	// Out is where prompts and notices are written (typically os.Stdout).
	Out io.Writer
}

// Result is a fetched template repository on local disk.
type Result struct {
	// Dir is the temporary directory containing the clone.
	Dir string
	// Commit is the commit ID the clone resolved to.
	Commit string
}

// Cleanup removes the temporary clone. It is safe to call more than once.
func (r *Result) Cleanup() {
	if r.Dir != "" {
		os.RemoveAll(r.Dir)
		r.Dir = ""
	}
}

// Fetch clones the template repository described by opts into a temporary
// directory. The caller is responsible for calling Cleanup on the result.
func Fetch(opts Options) (*Result, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("template URL cannot be empty")
	}
	if opts.In == nil {
		opts.In = os.Stdin
	}
	if opts.Out == nil {
		opts.Out = os.Stdout
	}

	auth, err := sshAuth(opts)
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "jig-template-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	repo, err := clone(tmpDir, auth, opts)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to resolve HEAD of cloned template repository: %w", err)
	}

	return &Result{Dir: tmpDir, Commit: head.Hash().String()}, nil
}

// sshAuth builds an ssh-agent auth method with host key verification for ssh
// URLs. For every other transport it returns nil so go-git uses its default
// (anonymous) behaviour.
func sshAuth(opts Options) (transport.AuthMethod, error) {
	ep, err := transport.NewEndpoint(opts.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid template URL %q: %w", opts.URL, err)
	}
	if ep.Protocol != "ssh" {
		return nil, nil
	}

	auth, err := gitssh.NewSSHAgentAuth(ep.User)
	if err != nil {
		return nil, fmt.Errorf("ssh template URLs require a running ssh-agent (is SSH_AUTH_SOCK set?): %w", err)
	}

	path, err := knownHostsPath()
	if err != nil {
		return nil, err
	}
	cb, err := hostKeyCallback(path, opts.SSHAcceptNew, opts.In, opts.Out)
	if err != nil {
		return nil, err
	}
	auth.HostKeyCallback = cb
	return auth, nil
}

// clone performs a shallow, single-branch clone of opts.URL into dir. A bare
// ref name is tried as a branch first and as a tag second, since the clone
// refspec needs a fully qualified name.
func clone(dir string, auth transport.AuthMethod, opts Options) (*git.Repository, error) {
	candidates := []plumbing.ReferenceName{""}
	switch {
	case opts.Ref == "":
		// Default branch; single zero candidate.
	case strings.HasPrefix(opts.Ref, "refs/"):
		candidates = []plumbing.ReferenceName{plumbing.ReferenceName(opts.Ref)}
	default:
		candidates = []plumbing.ReferenceName{
			plumbing.NewBranchReferenceName(opts.Ref),
			plumbing.NewTagReferenceName(opts.Ref),
		}
	}

	var lastErr error
	for i, refName := range candidates {
		if i > 0 {
			// A failed attempt leaves a partial clone behind; start over
			// with an empty directory.
			if err := os.RemoveAll(dir); err != nil {
				return nil, fmt.Errorf("failed to reset temporary directory: %w", err)
			}
			if err := os.MkdirAll(dir, 0700); err != nil {
				return nil, fmt.Errorf("failed to reset temporary directory: %w", err)
			}
		}

		repo, err := git.PlainClone(dir, false, &git.CloneOptions{
			URL:           opts.URL,
			Auth:          auth,
			ReferenceName: refName,
			SingleBranch:  true,
			Depth:         1,
		})
		if err == nil {
			return repo, nil
		}
		lastErr = err

		refNotFound := errors.Is(err, git.NoMatchingRefSpecError{}) ||
			errors.Is(err, plumbing.ErrReferenceNotFound)
		if !refNotFound {
			return nil, fmt.Errorf("failed to clone template repository %s: %w", opts.URL, err)
		}
	}

	if opts.Ref != "" {
		return nil, fmt.Errorf("ref %q not found in template repository %s: %w", opts.Ref, opts.URL, lastErr)
	}
	return nil, fmt.Errorf("failed to clone template repository %s: %w", opts.URL, lastErr)
}
