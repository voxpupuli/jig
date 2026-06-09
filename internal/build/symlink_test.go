package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withSymlinkSupport skips the test if the platform/filesystem cannot create
// symlinks (e.g. Windows without privilege). Keeps the suite green elsewhere.
func withSymlinkSupport(t *testing.T, dir string) {
	t.Helper()
	target := filepath.Join(dir, ".symlink_probe_target")
	link := filepath.Join(dir, ".symlink_probe_link")
	if err := os.WriteFile(target, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}
	_ = os.Remove(link)
	_ = os.Remove(target)
}

// symlink creates a symlink at dir/linkRel pointing at target (raw, may be
// relative or absolute, may dangle).
func symlink(t *testing.T, dir, linkRel, target string) {
	t.Helper()
	link := filepath.Join(dir, linkRel)
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns
// everything written. DoBuild reports warnings via fmt.Printf, so this is the
// only way to assert on warning behavior.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				sb.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		done <- sb.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = orig
	return <-done
}

// TestDoBuild_SymlinkToFileExcluded verifies a symlink pointing at a regular
// file inside the module is never written into the archive.
func TestDoBuild_SymlinkToFileExcluded(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")
	withSymlinkSupport(t, dir)

	// Real file that is included, plus a symlink to it.
	writeFile(t, dir, "files/real.txt", "hello")
	symlink(t, dir, "files/link.txt", filepath.Join(dir, "files/real.txt"))

	if err := DoBuild(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := archiveEntries(t, filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz"))
	if !containsEntry(entries, "files/real.txt") {
		t.Errorf("expected real file files/real.txt to be archived; entries: %v", entries)
	}
	if containsEntry(entries, "files/link.txt") {
		t.Errorf("symlink files/link.txt must not be in the archive; entries: %v", entries)
	}
}

// TestDoBuild_SymlinkToDirExcluded verifies a symlink pointing at a directory
// is skipped and, critically, that the walk does not descend through it and
// pull in the target's contents under the link's name.
func TestDoBuild_SymlinkToDirExcluded(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")
	withSymlinkSupport(t, dir)

	// A real directory with content, and a symlink to it.
	writeFile(t, dir, "templates/inner/file.erb", "<%= x %>")
	symlink(t, dir, "templates/linkdir", filepath.Join(dir, "templates/inner"))

	if err := DoBuild(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := archiveEntries(t, filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz"))

	// The real directory and its file should be present.
	if !containsEntry(entries, "templates/inner/file.erb") {
		t.Errorf("expected real templates/inner/file.erb in archive; entries: %v", entries)
	}
	// The symlinked dir must not appear, nor any content reached through it.
	for _, e := range entries {
		norm := filepath.ToSlash(e)
		if strings.Contains(norm, "templates/linkdir") {
			t.Errorf("symlinked dir templates/linkdir leaked into archive: %q", e)
		}
	}
}

// TestDoBuild_DanglingSymlink verifies a symlink whose target does not exist
// does not crash the build; it is warned and skipped like any other symlink.
func TestDoBuild_DanglingSymlink(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")
	withSymlinkSupport(t, dir)

	symlink(t, dir, "files/dangling.txt", filepath.Join(dir, "files/does-not-exist"))

	if err := DoBuild(dir); err != nil {
		t.Fatalf("dangling symlink should not error the build, got: %v", err)
	}

	entries := archiveEntries(t, filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz"))
	if containsEntry(entries, "files/dangling.txt") {
		t.Errorf("dangling symlink must not be archived; entries: %v", entries)
	}
}

// TestDoBuild_SymlinkWarns verifies a non-ignored symlink produces a warning
// on stdout that names the offending link.
func TestDoBuild_SymlinkWarns(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")
	withSymlinkSupport(t, dir)

	writeFile(t, dir, "files/real.txt", "hello")
	symlink(t, dir, "files/link.txt", filepath.Join(dir, "files/real.txt"))

	out := captureStdout(t, func() {
		if err := DoBuild(dir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "link.txt") || !strings.Contains(out, "symlink") {
		t.Errorf("expected a symlink warning naming link.txt, got output: %q", out)
	}
}

// TestDoBuild_IgnoredSymlinkSilent verifies a symlink that .pdkignore already
// excludes does NOT warn. The fixture ignores /spec/, so a symlink under spec/
// should be pruned silently. Precedence: ignore match wins over symlink check.
func TestDoBuild_IgnoredSymlinkSilent(t *testing.T) {
	dir := makeBuildDir(t, "myuser", "mymodule")
	withSymlinkSupport(t, dir)

	writeFile(t, dir, "files/real.txt", "hello")
	symlink(t, dir, "spec/link.txt", filepath.Join(dir, "files/real.txt"))

	out := captureStdout(t, func() {
		if err := DoBuild(dir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if strings.Contains(out, "link.txt") {
		t.Errorf("ignored symlink under spec/ should not warn, got output: %q", out)
	}

	entries := archiveEntries(t, filepath.Join(dir, "pkg", "myuser-mymodule-0.1.0.tar.gz"))
	if containsEntry(entries, "spec/link.txt") {
		t.Errorf("ignored symlink must not be archived; entries: %v", entries)
	}
}
