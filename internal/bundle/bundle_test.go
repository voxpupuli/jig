// SPDX-License-Identifier: GPL-3.0-or-later
package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// TestHelperProcess is not a real test. It is invoked as a subprocess by the
// fake execCommand below, standing in for the `bundle` binary. It inspects the
// args it was handed and the GO_HELPER_* environment to decide how to behave:
// echo argv, exit with a chosen code, or simulate a missing binary.
//
// Pattern per os/exec's own tests: the fake execCommand re-execs the test
// binary with -test.run=TestHelperProcess and GO_WANT_HELPER_PROCESS=1.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// Args after the "--" separator are the ones RunBundle actually passed
	// to execCommand (i.e. "bundle" + the bundle args).
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}

	// args[0] is the command name ("bundle"); args[1:] are the forwarded args.
	if mode := os.Getenv("GO_HELPER_MODE"); mode == "echo" {
		// Emit the full argv so the parent can assert exact fidelity.
		fmt.Fprint(os.Stdout, strings.Join(args, "\x1f"))
		os.Exit(0)
	}

	code, _ := strconv.Atoi(os.Getenv("GO_HELPER_EXIT"))
	os.Exit(code)
}

// fakeExec returns an execCommand replacement that re-execs this test binary
// into TestHelperProcess, threading the given env through.
func fakeExec(env map[string]string) func(string, ...string) *exec.Cmd {
	return func(name string, arg ...string) *exec.Cmd {
		cs := append([]string{"-test.run=TestHelperProcess", "--", name}, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
		return cmd
	}
}

// withSeams swaps execCommand/osExit for the duration of a test and restores
// them afterward, so cases can't bleed into one another.
func withSeams(t *testing.T, ec func(string, ...string) *exec.Cmd, ex func(int)) {
	t.Helper()
	origExec, origExit := execCommand, osExit
	execCommand, osExit = ec, ex
	t.Cleanup(func() { execCommand, osExit = origExec, origExit })
}

// --- Tests -----------------------------------------------------------------

// A clean (exit 0) child must return nil and must NOT call osExit. If osExit
// fires on the success path that's a serious regression (it would kill the
// real process on every successful command).
func TestRunBundle_SuccessReturnsNilAndDoesNotExit(t *testing.T) {
	exitCalled := false
	withSeams(t,
		fakeExec(map[string]string{"GO_HELPER_EXIT": "0"}),
		func(int) { exitCalled = true },
	)

	if err := RunBundle(Runner{}, []string{"exec", "rake", "spec"}); err != nil {
		t.Fatalf("expected nil error on clean exit, got %v", err)
	}
	if exitCalled {
		t.Fatal("osExit was called on the success path; it must not be")
	}
}

// A non-zero child exit must be propagated verbatim through osExit, not
// flattened to 1 and not swallowed. Probe several codes including the
// boundary-ish ones rake/rspec actually use.
func TestRunBundle_PropagatesChildExitCode(t *testing.T) {
	for _, code := range []int{1, 2, 5, 127, 255} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			var got int
			gotCalled := false
			withSeams(t,
				fakeExec(map[string]string{"GO_HELPER_EXIT": strconv.Itoa(code)}),
				func(c int) { got = c; gotCalled = true },
			)

			// RunBundle returns normally here only because our fake osExit
			// doesn't terminate; in production os.Exit would not return.
			_ = RunBundle(Runner{}, []string{"exec", "rake", "validate"})

			if !gotCalled {
				t.Fatalf("expected osExit to be called for exit code %d", code)
			}
			if got != code {
				t.Fatalf("exit code not propagated: child exited %d, osExit got %d", code, got)
			}
		})
	}
}

// When the binary genuinely cannot be started (not an *exec.ExitError but a
// startup/lookup failure), RunBundle must return the error rather than calling
// osExit. We simulate this with an execCommand that points at a path which
// cannot execute, so cmd.Run fails before any exit status exists.
func TestRunBundle_StartFailureReturnsErrorNotExit(t *testing.T) {
	exitCalled := false
	withSeams(t,
		func(name string, arg ...string) *exec.Cmd {
			// A path that does not exist -> exec fails at start, yielding a
			// non-ExitError from Run().
			return exec.Command("/nonexistent/jig-test-no-such-binary")
		},
		func(int) { exitCalled = true },
	)

	err := RunBundle(Runner{}, []string{"exec", "msync", "update"})
	if err == nil {
		t.Fatal("expected an error when the command cannot start, got nil")
	}
	if exitCalled {
		t.Fatal("osExit must not be called for a start/lookup failure; error should be returned")
	}
}

// Arg fidelity: every element RunBundle is given must reach the child exactly,
// in order, with "bundle" prepended and nothing re-split, dropped, or merged.
// This is the adversarial guard for the append/DisableFlagParsing bugs that
// bit the call sites: args containing spaces, leading dashes, and shell
// metacharacters must survive untouched because no shell is involved.
func TestRunBundle_PassesArgsVerbatim(t *testing.T) {
	cases := [][]string{
		{"exec", "rake", "spec"},
		{"exec", "rake", "validate", "lint"},
		{"exec", "msync", "update"},
		// adversarial payloads:
		{"exec", "rake", "spec", "SPEC=spec/has a space_spec.rb"},
		{"exec", "rake", "--trace"},
		{"exec", "rake", "spec", "FOO=a;rm -rf /", "BAR=$(whoami)"},
		{"exec", "rake", "spec", "--", "-d", "--debug"},
		{}, // empty: child should still receive just "bundle"
	}

	for i, args := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var captured string
			withSeams(t,
				func(name string, arg ...string) *exec.Cmd {
					cmd := fakeExec(map[string]string{"GO_HELPER_MODE": "echo"})(name, arg...)
					// Capture stdout from the helper into our buffer by
					// running it directly here instead of inheriting os.Stdout.
					out, _ := cmd.Output()
					captured = string(out)
					// Return a harmless command so RunBundle's own Run() is a
					// clean no-op; we've already captured what we need.
					return fakeExec(map[string]string{"GO_HELPER_EXIT": "0"})("bundle")
				},
				func(int) {},
			)

			_ = RunBundle(Runner{}, args)

			gotParts := []string{}
			if captured != "" {
				gotParts = strings.Split(captured, "\x1f")
			}
			want := append([]string{"bundle"}, args...)
			if len(gotParts) != len(want) {
				t.Fatalf("arg count mismatch:\n got  %#v\n want %#v", gotParts, want)
			}
			for j := range want {
				if gotParts[j] != want[j] {
					t.Fatalf("arg %d mismatch:\n got  %q\n want %q\n(full got=%#v want=%#v)",
						j, gotParts[j], want[j], gotParts, want)
				}
			}
		})
	}
}

// Guard that the command name is exactly "bundle" and never something derived
// from the args (e.g. a regression where args[0] got used as the binary).
func TestRunBundle_AlwaysInvokesBundle(t *testing.T) {
	var gotName string
	withSeams(t,
		func(name string, arg ...string) *exec.Cmd {
			gotName = name
			return fakeExec(map[string]string{"GO_HELPER_EXIT": "0"})("bundle")
		},
		func(int) {},
	)

	_ = RunBundle(Runner{}, []string{"exec", "rake", "spec"})
	if gotName != "bundle" {
		t.Fatalf("expected command name %q, got %q", "bundle", gotName)
	}
}

// --- Runner resolution -----------------------------------------------------

// withGetwd swaps the osGetwd seam so the container runner sees a deterministic
// module root, restoring it afterward.
func withGetwd(t *testing.T, fn func() (string, error)) {
	t.Helper()
	orig := osGetwd
	osGetwd = fn
	t.Cleanup(func() { osGetwd = orig })
}

// The local runner (zero value and explicit "local") must invoke the host's
// bundle directly. Rake tasks become `bundle exec rake <tasks>`; raw bundle
// args pass through untouched. Neither path consults osGetwd.
func TestRunner_LocalResolves(t *testing.T) {
	for _, typ := range []string{"", "local"} {
		t.Run("type="+typ, func(t *testing.T) {
			withGetwd(t, func() (string, error) {
				t.Fatal("local runner must not consult the working directory")
				return "", nil
			})

			name, args, err := Runner{Type: typ}.resolveRake([]string{"spec"})
			if err != nil {
				t.Fatalf("resolveRake: unexpected error: %v", err)
			}
			if name != "bundle" {
				t.Fatalf("resolveRake command: got %q, want %q", name, "bundle")
			}
			if want := []string{"exec", "rake", "spec"}; !equal(args, want) {
				t.Fatalf("resolveRake args mismatch:\n got  %#v\n want %#v", args, want)
			}

			name, args, err = Runner{Type: typ}.resolveBundle([]string{"exec", "msync", "update"})
			if err != nil {
				t.Fatalf("resolveBundle: unexpected error: %v", err)
			}
			if name != "bundle" {
				t.Fatalf("resolveBundle command: got %q, want %q", name, "bundle")
			}
			if want := []string{"exec", "msync", "update"}; !equal(args, want) {
				t.Fatalf("resolveBundle args mismatch:\n got  %#v\n want %#v", args, want)
			}
		})
	}
}

// A voxbox rake task relies on the image's rake entrypoint, so the task names
// are passed as the bare container command (no `bundle exec rake` prefix, no
// --entrypoint override). The module is mounted at /repo.
func TestRunner_VoxboxRakeInvocation(t *testing.T) {
	withGetwd(t, func() (string, error) { return "/home/user/mymodule", nil })

	name, args, err := Runner{Type: "voxbox"}.resolveRake([]string{"validate", "lint"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != defaultEngine {
		t.Fatalf("expected engine %q, got %q", defaultEngine, name)
	}
	want := []string{
		"run", "--rm", "-i",
		"-v", "/home/user/mymodule:/repo:Z",
		"-w", "/repo",
		defaultImage,
		"validate", "lint",
	}
	if !equal(args, want) {
		t.Fatalf("container argv mismatch:\n got  %#v\n want %#v", args, want)
	}
}

// A voxbox raw bundle command must override the image's rake entrypoint back to
// `bundle` (e.g. for msync, which is not a rake task). Custom engine and image
// must be honoured verbatim; "container" is an alias for "voxbox".
func TestRunner_VoxboxBundleInvocation(t *testing.T) {
	withGetwd(t, func() (string, error) { return "/repo/path", nil })

	name, args, err := Runner{
		Type:   "container",
		Engine: "podman",
		Image:  "localhost/voxbox:dev",
	}.resolveBundle([]string{"exec", "msync", "update"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "podman" {
		t.Fatalf("expected engine %q, got %q", "podman", name)
	}
	want := []string{
		"run", "--rm", "-i",
		"-v", "/repo/path:/repo:Z",
		"-w", "/repo",
		"--entrypoint", "bundle",
		"localhost/voxbox:dev",
		"exec", "msync", "update",
	}
	if !equal(args, want) {
		t.Fatalf("container argv mismatch:\n got  %#v\n want %#v", args, want)
	}
}

// An unrecognised runner type must be a hard error on both paths, not a silent
// fallback to the local bundle (which would defeat opting into a container).
func TestRunner_UnknownTypeErrors(t *testing.T) {
	if _, _, err := (Runner{Type: "nope"}).resolveRake([]string{"spec"}); err == nil {
		t.Fatal("resolveRake: expected an error for an unknown runner type, got nil")
	}
	if _, _, err := (Runner{Type: "nope"}).resolveBundle([]string{"exec", "msync", "update"}); err == nil {
		t.Fatal("resolveBundle: expected an error for an unknown runner type, got nil")
	}
}

// If the working directory can't be determined, the container runner must
// surface that as an error rather than mounting a bogus path.
func TestRunner_VoxboxGetwdFailureErrors(t *testing.T) {
	withGetwd(t, func() (string, error) { return "", fmt.Errorf("boom") })

	if _, _, err := (Runner{Type: "voxbox"}).resolveRake([]string{"spec"}); err == nil {
		t.Fatal("resolveRake: expected an error when the working directory is unavailable, got nil")
	}
	if _, _, err := (Runner{Type: "voxbox"}).resolveBundle([]string{"exec", "msync", "update"}); err == nil {
		t.Fatal("resolveBundle: expected an error when the working directory is unavailable, got nil")
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
