// SPDX-License-Identifier: GPL-3.0-or-later
package build

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	gogitignore "github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/voxpupuli/jig/internal/module"
	"github.com/voxpupuli/jig/internal/scaffold"
)

func DoBuild(dir string) error {
	metadata, err := scaffold.GetMetadata(dir)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	results := metadata.Validate()
	for _, result := range results {
		if result.Level == module.Error {
			return fmt.Errorf("metadata validation failed: %s - %s", result.Field, result.Message)
		}
		if result.Level == module.Warning {
			fmt.Printf("warning: %s - %s\n", result.Field, result.Message)
		}
	}

	// Read the ignore file (.pdkignore, .pmtignore, or .gitignore, in that
	// order) and parse patterns line by line
	ignoreData, _, err := readIgnoreFile(dir)
	if err != nil {
		return err
	}

	var patterns []gogitignore.Pattern
	for _, line := range strings.Split(string(ignoreData), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, gogitignore.ParsePattern(line, nil))
	}

	patterns = append(patterns, gogitignore.ParsePattern(".gitkeep", nil))
	patterns = append(patterns, gogitignore.ParsePattern(".rubocop.yml", nil))

	matcher := gogitignore.NewMatcher(patterns)

	pkgDir := filepath.Join(dir, "pkg")
	if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
		if err = os.MkdirAll(pkgDir, 0755); err != nil {
			return fmt.Errorf("failed to create pkg directory: %w", err)
		}
	}

	archiveName := fmt.Sprintf("%s-%s-%s.tar.gz", metadata.ForgeUsername(), metadata.ModuleName(), metadata.Version)
	archivePath := filepath.Join(pkgDir, archiveName)

	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer archiveFile.Close()

	gzWriter := gzip.NewWriter(archiveFile)
	defer gzWriter.Close()
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	prefix := fmt.Sprintf("%s-%s-%s", metadata.ForgeUsername(), metadata.ModuleName(), metadata.Version)

	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		parts := strings.Split(filepath.ToSlash(relPath), "/")

		if matcher.Match(parts, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip symlinks
		if d.Type()&fs.ModeSymlink != 0 {
			fmt.Printf("warning: %s is a symlink, skipping\n", d.Name())
			return nil
		}

		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return fmt.Errorf("failed to create tar header for %s: %w", path, err)
			}
			header.Name = filepath.Join(prefix, relPath)
			return tarWriter.WriteHeader(header)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", path, err)
		}

		header.Name = filepath.Join(prefix, relPath)

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %w", path, err)
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer f.Close()

		if _, err = io.Copy(tarWriter, f); err != nil {
			return fmt.Errorf("failed to write file %s to archive: %w", path, err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk module directory: %w", err)
	}

	fmt.Printf("built %s\n", archivePath)
	return nil
}
