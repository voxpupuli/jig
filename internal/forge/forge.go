// SPDX-License-Identifier: GPL-3.0-or-later
package forge

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

const forgeAPIURL = "https://forgeapi.puppet.com/v3/releases"

// Publisher uploads a built module archive to the Puppet Forge.
type Publisher interface {
	Publish(archivePath string) error
}

type httpPublisher struct {
	token  string
	client *http.Client
}

// NewPublisher returns a Publisher that uploads to the Puppet Forge using
// the provided API token.
func NewPublisher(token string) Publisher {
	return &httpPublisher{
		token:  token,
		client: &http.Client{},
	}
}

func (p *httpPublisher) Publish(archivePath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive for upload: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	part, err := w.CreateFormFile("file", filepath.Base(archivePath))
	if err != nil {
		return fmt.Errorf("failed to create multipart field: %w", err)
	}

	if _, err = io.Copy(part, f); err != nil {
		return fmt.Errorf("failed to write archive to multipart body: %w", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("failed to finalise multipart body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, forgeAPIURL, &body)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+p.token)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("forge returned unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
