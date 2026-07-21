// SPDX-License-Identifier: GPL-3.0-or-later
package remote

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// fixtureRepo builds a local template repository with a known shape: the
// default branch (master) holds hello.txt at commit "master", a "feature"
// branch adds feature.txt at commit "feature", and tag "v1" points at the
// master commit. It returns the repo path and the two commit IDs.
func fixtureRepo(t *testing.T) (string, map[string]string) {
	t.Helper()
	dir := t.TempDir()

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init fixture repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("fixture worktree: %v", err)
	}

	sig := &object.Signature{Name: "fixture", Email: "fixture@example.com", When: time.Now()}

	commit := func(file, message string) plumbing.Hash {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, file), []byte(message), 0o644); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
		if _, err := wt.Add(file); err != nil {
			t.Fatalf("add %s: %v", file, err)
		}
		hash, err := wt.Commit(message, &git.CommitOptions{Author: sig})
		if err != nil {
			t.Fatalf("commit %s: %v", message, err)
		}
		return hash
	}

	master := commit("hello.txt", "master")
	if _, err := repo.CreateTag("v1", master, nil); err != nil {
		t.Fatalf("create tag: %v", err)
	}

	err = wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("feature"),
		Create: true,
	})
	if err != nil {
		t.Fatalf("checkout feature: %v", err)
	}
	feature := commit("feature.txt", "feature")

	err = wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("master")})
	if err != nil {
		t.Fatalf("checkout master: %v", err)
	}

	return dir, map[string]string{
		"master":  master.String(),
		"feature": feature.String(),
	}
}

func mustFetch(t *testing.T, opts Options) *Result {
	t.Helper()
	res, err := Fetch(opts)
	if err != nil {
		t.Fatalf("Fetch(%+v): %v", opts, err)
	}
	t.Cleanup(res.Cleanup)
	return res
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// With no ref, Fetch must clone the remote's default branch and report the
// commit it resolved to, so metadata.json can record the exact template state.
func TestFetch_DefaultBranch(t *testing.T) {
	url, commits := fixtureRepo(t)

	res := mustFetch(t, Options{URL: url})

	if !fileExists(filepath.Join(res.Dir, "hello.txt")) {
		t.Error("expected hello.txt from the default branch")
	}
	if fileExists(filepath.Join(res.Dir, "feature.txt")) {
		t.Error("feature.txt from another branch should not be present")
	}
	if res.Commit != commits["master"] {
		t.Errorf("commit: got %s, want %s", res.Commit, commits["master"])
	}
}

// A bare ref name that exists as a branch must check that branch out.
func TestFetch_BranchRef(t *testing.T) {
	url, commits := fixtureRepo(t)

	res := mustFetch(t, Options{URL: url, Ref: "feature"})

	if !fileExists(filepath.Join(res.Dir, "feature.txt")) {
		t.Error("expected feature.txt from the feature branch")
	}
	if res.Commit != commits["feature"] {
		t.Errorf("commit: got %s, want %s", res.Commit, commits["feature"])
	}
}

// A bare ref name that only exists as a tag must fall back to the tag after
// the branch attempt misses.
func TestFetch_TagRef(t *testing.T) {
	url, commits := fixtureRepo(t)

	res := mustFetch(t, Options{URL: url, Ref: "v1"})

	if res.Commit != commits["master"] {
		t.Errorf("commit: got %s, want %s", res.Commit, commits["master"])
	}
}

// A fully qualified ref name must be used verbatim, with no branch/tag
// guessing.
func TestFetch_FullyQualifiedRef(t *testing.T) {
	url, commits := fixtureRepo(t)

	res := mustFetch(t, Options{URL: url, Ref: "refs/heads/feature"})

	if res.Commit != commits["feature"] {
		t.Errorf("commit: got %s, want %s", res.Commit, commits["feature"])
	}
}

// A ref that exists neither as a branch nor as a tag must fail with an error
// naming the ref, and must not leave the temporary directory behind.
func TestFetch_MissingRef(t *testing.T) {
	url, _ := fixtureRepo(t)

	res, err := Fetch(Options{URL: url, Ref: "does-not-exist"})
	if err == nil {
		res.Cleanup()
		t.Fatal("expected an error for a missing ref")
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Errorf("error should name the missing ref, got: %v", err)
	}
}

func TestFetch_EmptyURL(t *testing.T) {
	if _, err := Fetch(Options{}); err == nil {
		t.Fatal("expected an error for an empty URL")
	}
}

// Cleanup must remove the temporary clone and stay safe when called twice.
func TestResult_Cleanup(t *testing.T) {
	url, _ := fixtureRepo(t)

	res, err := Fetch(Options{URL: url})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	dir := res.Dir
	res.Cleanup()
	if fileExists(dir) {
		t.Errorf("temporary directory %s still exists after Cleanup", dir)
	}
	res.Cleanup() // must not panic
}
