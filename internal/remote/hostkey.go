// SPDX-License-Identifier: GPL-3.0-or-later
package remote

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// knownHostsPath returns the known_hosts file to verify against:
// $SSH_KNOWN_HOSTS if set, otherwise ~/.ssh/known_hosts.
func knownHostsPath() (string, error) {
	if p := os.Getenv("SSH_KNOWN_HOSTS"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "known_hosts"), nil
}

// hostKeyCallback verifies server keys against the known_hosts file at path,
// reproducing OpenSSH's StrictHostKeyChecking behaviour: "ask" by default,
// "accept-new" when acceptNew is set. A key that differs from the recorded
// one is always a hard error; there is deliberately no way to override that.
func hostKeyCallback(path string, acceptNew bool, in io.Reader, out io.Writer) (gossh.HostKeyCallback, error) {
	if err := ensureKnownHostsFile(path); err != nil {
		return nil, err
	}
	kh, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load known hosts from %s: %w", path, err)
	}

	return func(hostname string, remote net.Addr, key gossh.PublicKey) error {
		err := kh(hostname, remote, key)
		if err == nil {
			return nil
		}

		var keyErr *knownhosts.KeyError
		if !errors.As(err, &keyErr) {
			return err
		}

		if len(keyErr.Want) > 0 {
			return fmt.Errorf(
				"host key verification failed: the %s key for %s does not match the one recorded in %s.\n"+
					"This can mean the server was reinstalled, or that the connection is being intercepted.\n"+
					"If you are certain the new key is legitimate, remove the stale entry (e.g. `ssh-keygen -R %s`) and try again: %w",
				key.Type(), hostname, path, knownhosts.Normalize(hostname), err)
		}

		fingerprint := gossh.FingerprintSHA256(key)

		if acceptNew {
			fmt.Fprintf(out, "Accepting new %s host key for %s (%s) and adding it to %s.\n",
				key.Type(), hostname, fingerprint, path)
			return appendKnownHost(path, hostname, key)
		}

		if !isTerminalFn(in) {
			return fmt.Errorf(
				"the authenticity of host %q can't be established (%s key fingerprint %s) and there is no terminal to ask on.\n"+
					"Re-run interactively, pass --ssh-accept-new, or connect to the host once with ssh first: %w",
				hostname, key.Type(), fingerprint, err)
		}

		fmt.Fprintf(out, "The authenticity of host %q can't be established.\n", hostname)
		fmt.Fprintf(out, "%s key fingerprint is %s.\n", key.Type(), fingerprint)
		fmt.Fprint(out, "Are you sure you want to continue connecting (yes/no)? ")

		scanner := bufio.NewScanner(in)
		if !scanner.Scan() {
			return fmt.Errorf("host key for %s was not accepted: %w", hostname, err)
		}
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if answer != "yes" && answer != "y" {
			return fmt.Errorf("host key for %s was not accepted: %w", hostname, err)
		}

		if appendErr := appendKnownHost(path, hostname, key); appendErr != nil {
			return appendErr
		}
		fmt.Fprintf(out, "Warning: permanently added %q (%s) to the list of known hosts.\n",
			hostname, key.Type())
		return nil
	}, nil
}

// ensureKnownHostsFile creates the known_hosts file (and its directory) with
// OpenSSH's usual permissions if it does not exist yet, so that a fresh
// machine behaves like any other unknown-host case instead of erroring.
func ensureKnownHostsFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	return f.Close()
}

func appendKnownHost(path string, hostname string, key gossh.PublicKey) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	line := knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key)
	if _, err := f.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("failed to update %s: %w", path, err)
	}
	return nil
}

// isTerminalFn is a variable so tests can simulate an interactive terminal.
var isTerminalFn = isTerminal

// isTerminal reports whether in is an interactive terminal. Anything that is
// not an *os.File (e.g. a test buffer) is treated as non-interactive.
func isTerminal(in io.Reader) bool {
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
