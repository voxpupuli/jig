// SPDX-License-Identifier: GPL-3.0-or-later
package bundle

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// indirection seams for testing
var (
	execCommand = exec.Command
	osExit      = os.Exit
	osGetwd     = os.Getwd
)

// Default values for the container runner, mirrored from the config package so
// the bundle layer can stand on its own (and be tested without importing it).
const (
	defaultEngine = "docker"
	defaultImage  = "ghcr.io/voxpupuli/voxbox:latest"
)

// Runner describes how a bundle-backed command should be executed. The zero
// value (or Type "local") runs the host's `bundle` directly; Type "voxbox"
// runs inside the voxbox container so no system-wide Ruby/bundler is needed.
//
// The voxbox image is opinionated: its entrypoint is already
// `bundle exec rake -f <voxbox Rakefile>` with the workdir at /repo. So rake
// tasks (RunRake) are passed as bare task names, while raw `bundle` commands
// (RunBundle) override the entrypoint back to `bundle`.
type Runner struct {
	Type   string
	Engine string
	Image  string
}

// RunRake runs the given rake task(s) against the current module. Locally this
// is `bundle exec rake <args>`; in voxbox it relies on the image's rake
// entrypoint, passing the task names through.
func RunRake(r Runner, rakeArgs []string) error {
	name, args, err := r.resolveRake(rakeArgs)
	if err != nil {
		return err
	}
	return run(name, args)
}

// RunBundle runs a raw `bundle <args>` command. Locally that is the host
// bundle; in voxbox the entrypoint is overridden back to `bundle` (the image
// otherwise defaults to running rake).
func RunBundle(r Runner, bundleArgs []string) error {
	name, args, err := r.resolveBundle(bundleArgs)
	if err != nil {
		return err
	}
	return run(name, args)
}

func (r Runner) resolveRake(rakeArgs []string) (string, []string, error) {
	switch r.Type {
	case "", "local":
		return "bundle", append([]string{"exec", "rake"}, rakeArgs...), nil
	case "voxbox", "container":
		// The voxbox entrypoint already wraps `bundle exec rake`, so the task
		// names are the container command.
		return r.container(nil, rakeArgs)
	default:
		return "", nil, unknownTypeErr(r.Type)
	}
}

func (r Runner) resolveBundle(bundleArgs []string) (string, []string, error) {
	switch r.Type {
	case "", "local":
		return "bundle", bundleArgs, nil
	case "voxbox", "container":
		// Override the image's rake entrypoint so we can invoke bundle directly
		// (e.g. for `bundle exec msync update`, which is not a rake task).
		return r.container([]string{"--entrypoint", "bundle"}, bundleArgs)
	default:
		return "", nil, unknownTypeErr(r.Type)
	}
}

// container builds the engine invocation that mounts the current module at
// /repo and runs cmd inside the image. extraOpts are passed to the engine
// before the image name (e.g. --entrypoint).
func (r Runner) container(extraOpts, cmd []string) (string, []string, error) {
	cwd, err := osGetwd()
	if err != nil {
		return "", nil, fmt.Errorf("could not determine working directory for container runner: %w", err)
	}
	engine := r.Engine
	if engine == "" {
		engine = defaultEngine
	}
	image := r.Image
	if image == "" {
		image = defaultImage
	}
	// Mount the module at /repo and run there. The :Z label is a no-op on
	// non-SELinux hosts but required for podman on SELinux systems (the
	// voxpupuli norm). -i keeps stdin open for prompts.
	args := []string{"run", "--rm", "-i", "-v", cwd + ":/repo:Z", "-w", "/repo"}
	args = append(args, extraOpts...)
	args = append(args, image)
	args = append(args, cmd...)
	return engine, args, nil
}

func unknownTypeErr(typ string) error {
	return fmt.Errorf("unknown runner type %q (expected \"local\" or \"voxbox\")", typ)
}

// run executes name with the given args, wiring through the standard streams. A
// non-zero child exit is propagated verbatim via os.Exit; a failure to start
// the command is returned as an error.
func run(name string, args []string) error {
	cmd := execCommand(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			osExit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}
