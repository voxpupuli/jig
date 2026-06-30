// SPDX-License-Identifier: GPL-3.0-or-later
package release

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/voxpupuli/jig/internal/build"
	"github.com/voxpupuli/jig/internal/forge"
	"github.com/voxpupuli/jig/internal/module"
	"github.com/voxpupuli/jig/internal/scaffold"
)

// Options controls the behaviour of DoRelease.
type Options struct {
	Version        string
	SkipValidation bool
	SkipBuild      bool
	SkipPublish    bool
}

var validVersionRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// DoRelease runs the full release sequence for the module in dir:
//  1. Validate the requested version string (unless SkipValidation).
//  2. Validate module metadata (unless SkipValidation).
//  3. Write the new version into metadata.json.
//  4. Build the module archive (unless SkipBuild).
//  5. Publish the archive to the Forge (unless SkipPublish).
func DoRelease(dir string, opts Options, publisher forge.Publisher) error {
	if !opts.SkipValidation {
		if !validVersionRe.MatchString(opts.Version) {
			return fmt.Errorf("invalid version %q: must be a valid semver string (MAJOR.MINOR.PATCH)", opts.Version)
		}

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
	}

	// Write version into metadata.json regardless of SkipValidation, since
	// this is the whole point of the release command.
	metadata, err := scaffold.GetMetadata(dir)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	metadata.Version = opts.Version

	metadataPath := filepath.Join(dir, "metadata.json")
	if err := metadata.Write(metadataPath); err != nil {
		return fmt.Errorf("failed to write metadata.json: %w", err)
	}

	fmt.Printf("set version to %s in metadata.json\n", opts.Version)

	if !opts.SkipBuild {
		if err := build.DoBuild(dir); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
	}

	if !opts.SkipPublish {
		archiveName := fmt.Sprintf("%s-%s-%s.tar.gz", metadata.ForgeUsername(), metadata.ModuleName(), opts.Version)
		archivePath := filepath.Join(dir, "pkg", archiveName)

		if _, err := os.Stat(archivePath); os.IsNotExist(err) {
			return fmt.Errorf("archive %s not found: run without --skip-build or run jig build first", archivePath)
		}

		fmt.Printf("publishing %s to the Forge...\n", archiveName)

		if err := publisher.Publish(archivePath); err != nil {
			return fmt.Errorf("publish failed: %w", err)
		}

		fmt.Printf("published %s\n", archiveName)
	}

	return nil
}
