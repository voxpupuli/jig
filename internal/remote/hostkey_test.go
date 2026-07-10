// SPDX-License-Identifier: GPL-3.0-or-later
package remote

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const testHost = "git.example.com:22"

var testAddr = &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 22}

func genKey(t *testing.T) gossh.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	key, err := gossh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("wrap key: %v", err)
	}
	return key
}

// knownHostsFile writes a known_hosts file recording the given keys for
// testHost and returns its path. With no keys, the file is empty.
func knownHostsFile(t *testing.T, keys ...gossh.PublicKey) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "known_hosts")
	var lines []string
	for _, key := range keys {
		lines = append(lines, knownhosts.Line([]string{knownhosts.Normalize(testHost)}, key))
	}
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write known_hosts: %v", err)
	}
	return path
}

// fakeTerminal makes the callback treat stdin as interactive for the duration
// of the test, so the prompt path can be exercised with a plain reader.
func fakeTerminal(t *testing.T) {
	t.Helper()
	isTerminalFn = func(io.Reader) bool { return true }
	t.Cleanup(func() { isTerminalFn = isTerminal })
}

func callback(t *testing.T, path string, acceptNew bool, in io.Reader, out io.Writer) gossh.HostKeyCallback {
	t.Helper()
	if in == nil {
		in = strings.NewReader("")
	}
	if out == nil {
		out = io.Discard
	}
	cb, err := hostKeyCallback(path, acceptNew, in, out)
	if err != nil {
		t.Fatalf("hostKeyCallback: %v", err)
	}
	return cb
}

// A host already recorded in known_hosts must verify silently.
func TestHostKey_KnownHost(t *testing.T) {
	key := genKey(t)
	path := knownHostsFile(t, key)

	cb := callback(t, path, false, nil, nil)
	if err := cb(testHost, testAddr, key); err != nil {
		t.Errorf("known host should verify, got: %v", err)
	}
}

// An unknown host with accept-new set must be trusted, recorded in
// known_hosts under the normalized name, and announced on the output.
func TestHostKey_UnknownHostAcceptNew(t *testing.T) {
	key := genKey(t)
	path := knownHostsFile(t)
	var out bytes.Buffer

	cb := callback(t, path, true, nil, &out)
	if err := cb(testHost, testAddr, key); err != nil {
		t.Fatalf("accept-new should trust an unknown host, got: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read known_hosts: %v", err)
	}
	if !strings.Contains(string(content), "git.example.com") {
		t.Errorf("known_hosts should record the host, got: %q", content)
	}
	if !strings.Contains(out.String(), gossh.FingerprintSHA256(key)) {
		t.Errorf("output should announce the accepted fingerprint, got: %q", out.String())
	}
}

// An unknown host without a terminal to ask on must fail and point at the
// available remedies rather than hanging on a prompt.
func TestHostKey_UnknownHostNonInteractive(t *testing.T) {
	key := genKey(t)
	path := knownHostsFile(t)

	cb := callback(t, path, false, strings.NewReader("yes\n"), nil)
	err := cb(testHost, testAddr, key)
	if err == nil {
		t.Fatal("unknown host without a terminal should fail")
	}
	if !strings.Contains(err.Error(), "--ssh-accept-new") {
		t.Errorf("error should mention --ssh-accept-new, got: %v", err)
	}
}

// Interactively answering yes must trust the host and record it; the prompt
// must show the fingerprint first.
func TestHostKey_PromptAccepted(t *testing.T) {
	fakeTerminal(t)
	key := genKey(t)
	path := knownHostsFile(t)
	var out bytes.Buffer

	cb := callback(t, path, false, strings.NewReader("yes\n"), &out)
	if err := cb(testHost, testAddr, key); err != nil {
		t.Fatalf("accepted prompt should trust the host, got: %v", err)
	}

	if !strings.Contains(out.String(), gossh.FingerprintSHA256(key)) {
		t.Errorf("prompt should show the fingerprint, got: %q", out.String())
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read known_hosts: %v", err)
	}
	if !strings.Contains(string(content), "git.example.com") {
		t.Errorf("known_hosts should record the host, got: %q", content)
	}
}

// Interactively declining must fail and leave known_hosts untouched.
func TestHostKey_PromptDeclined(t *testing.T) {
	fakeTerminal(t)
	key := genKey(t)
	path := knownHostsFile(t)

	cb := callback(t, path, false, strings.NewReader("no\n"), nil)
	if err := cb(testHost, testAddr, key); err == nil {
		t.Fatal("declined prompt should fail")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read known_hosts: %v", err)
	}
	if len(content) != 0 {
		t.Errorf("known_hosts should stay empty after declining, got: %q", content)
	}
}

// A key that differs from the recorded one is the potential-MITM case: it
// must fail hard even with accept-new set and a willing user on stdin, and
// known_hosts must not change.
func TestHostKey_MismatchAlwaysFails(t *testing.T) {
	fakeTerminal(t)
	recorded := genKey(t)
	offered := genKey(t)
	path := knownHostsFile(t, recorded)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read known_hosts: %v", err)
	}

	for _, acceptNew := range []bool{false, true} {
		cb := callback(t, path, acceptNew, strings.NewReader("yes\n"), nil)
		err := cb(testHost, testAddr, offered)
		if err == nil {
			t.Fatalf("acceptNew=%v: a changed host key must fail", acceptNew)
		}
		if !strings.Contains(err.Error(), "does not match") {
			t.Errorf("acceptNew=%v: error should flag the mismatch, got: %v", acceptNew, err)
		}
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read known_hosts: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Error("known_hosts must not change on a mismatch")
	}
}

// A missing known_hosts file (fresh machine) must be created rather than
// erroring, so first contact behaves like any other unknown host.
func TestHostKey_MissingKnownHostsFile(t *testing.T) {
	key := genKey(t)
	path := filepath.Join(t.TempDir(), ".ssh", "known_hosts")

	cb := callback(t, path, true, nil, nil)
	if err := cb(testHost, testAddr, key); err != nil {
		t.Fatalf("missing known_hosts should be created, got: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("known_hosts should exist afterwards: %v", err)
	}
}
