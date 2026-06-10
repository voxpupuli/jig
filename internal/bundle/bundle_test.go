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

	if err := RunBundle([]string{"exec", "rake", "spec"}); err != nil {
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
			_ = RunBundle([]string{"exec", "rake", "validate"})

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

	err := RunBundle([]string{"exec", "msync", "update"})
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

			_ = RunBundle(args)

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

	_ = RunBundle([]string{"exec", "rake", "spec"})
	if gotName != "bundle" {
		t.Fatalf("expected command name %q, got %q", "bundle", gotName)
	}
}
