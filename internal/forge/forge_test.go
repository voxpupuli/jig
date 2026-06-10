// SPDX-License-Identifier: GPL-3.0-or-later
package forge

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestPublish_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
		}
		if _, _, err := r.FormFile("file"); err != nil {
			t.Errorf("expected 'file' field in form: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	archivePath := writeTestArchive(t)

	p := &httpPublisher{
		token: "test-token",
		client: &http.Client{
			Transport: &redirectTransport{target: server.URL},
		},
	}

	if err := p.Publish(archivePath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublish_NonCreatedStatus(t *testing.T) {
	cases := []struct {
		name   string
		status int
	}{
		{"conflict", http.StatusConflict},
		{"unauthorized", http.StatusUnauthorized},
		{"server error", http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprintf(w, "error from forge")
			}))
			defer server.Close()

			archivePath := writeTestArchive(t)

			p := &httpPublisher{
				token: "test-token",
				client: &http.Client{
					Transport: &redirectTransport{target: server.URL},
				},
			}

			err := p.Publish(archivePath)
			if err == nil {
				t.Errorf("expected error for status %d, got nil", tc.status)
			}
		})
	}
}

func TestPublish_MissingArchive(t *testing.T) {
	p := NewPublisher("test-token").(*httpPublisher)
	err := p.Publish("/nonexistent/path/module.tar.gz")
	if err == nil {
		t.Error("expected error for missing archive, got nil")
	}
}

// redirectTransport rewrites every request URL to point at target,
// preserving the path and query. This lets tests use httptest servers
// without modifying production constants.
type redirectTransport struct {
	target string
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target, err := url.Parse(rt.target)
	if err != nil {
		return nil, fmt.Errorf("redirectTransport: invalid target %q: %w", rt.target, err)
	}
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = target.Scheme
	req2.URL.Host = target.Host
	return http.DefaultTransport.RoundTrip(req2)
}

func writeTestArchive(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "author-module-1.0.0.tar.gz")
	if err := os.WriteFile(path, []byte("fake archive contents"), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
