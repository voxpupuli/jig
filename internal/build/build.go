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

	"github.com/voxpupuli/jig/internal/config"
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

	moduleConfig, err := config.LoadModuleConfig(dir)
	if err != nil {
		return err
	}
	filter, err := newBuildFilter(moduleConfig.Build)
	if err != nil {
		return err
	}

	for _, ignoreFile := range findIgnoreFiles(dir) {
		fmt.Printf("warning: %s is not obeyed; packaging is controlled by the [build] section of jig.toml. Remove the file to avoid confusion.\n",
			filepath.Base(ignoreFile))
	}

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

	// Directory headers are written lazily, only for ancestors of files that
	// made it into the archive. writtenDirs tracks which ones exist already.
	writtenDirs := map[string]bool{}
	writeAncestorDirs := func(relPath string) error {
		parts := strings.Split(relPath, "/")
		for i := 1; i < len(parts); i++ {
			dirPath := strings.Join(parts[:i], "/")
			if writtenDirs[dirPath] {
				continue
			}
			info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(dirPath)))
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return fmt.Errorf("failed to create tar header for %s: %w", dirPath, err)
			}
			header.Name = filepath.Join(prefix, filepath.FromSlash(dirPath))
			if err := tarWriter.WriteHeader(header); err != nil {
				return fmt.Errorf("failed to write tar header for %s: %w", dirPath, err)
			}
			writtenDirs[dirPath] = true
		}
		return nil
	}

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

		slashPath := filepath.ToSlash(relPath)

		if d.IsDir() {
			// pkg (which we are writing into) and .git never belong in the
			// archive; skipping them here avoids walking their contents at
			// all. Other directories are neither included nor excluded on
			// their own -- headers are emitted for ancestors of included
			// files.
			if slashPath == "pkg" || slashPath == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		if !filter.Include(slashPath) {
			return nil
		}

		// Skip symlinks. Checked after the filter so that excluded symlinks
		// stay silent.
		if d.Type()&fs.ModeSymlink != 0 {
			fmt.Printf("warning: %s is a symlink, skipping\n", d.Name())
			return nil
		}

		if err := writeAncestorDirs(slashPath); err != nil {
			return err
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
