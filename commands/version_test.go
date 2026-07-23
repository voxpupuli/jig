// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"bytes"
	"strings"
	"testing"
)

// The stamped version wins over any build-info fallback, and the version
// command prints it prefixed with the binary name.
func TestVersionCmd(t *testing.T) {
	orig := version
	version = "1.2.3"
	defer func() { version = orig }()

	cmd := NewApp().versionCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != "jig 1.2.3" {
		t.Fatalf("version output = %q, want %q", got, "jig 1.2.3")
	}
}

// Without a stamped version and without usable build info the fallback is
// "dev" — never an empty string.
func TestVersionFallback(t *testing.T) {
	orig := version
	version = ""
	defer func() { version = orig }()

	if got := Version(); got == "" {
		t.Fatal("Version() returned an empty string")
	}
}
