// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/voxpupuli/jig/v2/internal/config"
	"github.com/voxpupuli/jig/v2/internal/module"
	"github.com/voxpupuli/jig/v2/internal/template"
)

// staleLockFiles are lockfiles tied to the Gemfile convert just replaced --
// they describe a dependency set that no longer matches it, so the next
// `bundle install` would trip over a stale lock rather than resolving fresh.
var staleLockFiles = []string{"Gemfile.lock"}

// pdkArtifacts are PDK-era files convert doesn't touch, because jig can't
// tell whether they still hold customization worth keeping (.rubocop.yml,
// .puppet-lint.rc) -- convert only warns that they're no longer needed.
var pdkArtifacts = []string{".sync.yml", ".pdkignore", ".rubocop.yml", ".puppet-lint.rc", ".fixtures.yml", ".vscode"}

// ConvertOptions controls ConvertModule. The metadata fields are only used
// when metadata.json is missing from TargetDir, to seed the one jig creates;
// when metadata.json already exists (valid or repairable) they are ignored.
type ConvertOptions struct {
	TargetDir string
	DryRun    bool
	// Out receives progress messages; defaults to os.Stdout.
	Out io.Writer

	ForgeUser    string
	Name         string
	Author       string
	License      string
	Summary      string
	Source       string
	ProjectPage  string
	Version      string
	Dependencies []module.Dependency

	// HasModulefile records whether the options above were pre-filled from a
	// Modulefile, so ConvertModule can warn that it's no longer needed.
	HasModulefile bool
}

// ConvertModule brings an existing module onto the toolchain jig's other
// commands expect: it ensures metadata.json exists and is at least minimally
// valid (creating or repairing it as needed), ensures jig.toml exists, and
// (re)writes Gemfile, Rakefile, and spec/spec_helper.rb from jig's embedded
// templates.
func ConvertModule(opts ConvertOptions) error {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}

	metadataPath := filepath.Join(opts.TargetDir, "metadata.json")
	switch _, err := os.Stat(metadataPath); {
	case err == nil:
		if err := repairMetadata(metadataPath, opts.DryRun, out); err != nil {
			return err
		}
	case os.IsNotExist(err):
		if err := createMetadata(opts, metadataPath, out); err != nil {
			return err
		}
	default:
		return fmt.Errorf("failed to stat %s: %w", metadataPath, err)
	}

	// jig.toml is jig's own config file, independent of whether
	// metadata.json needed to be created or repaired -- a module with a
	// perfectly valid but pre-jig.toml metadata.json needs one too, and
	// repair's own "move template-url to [template] in jig.toml" warning
	// only makes sense if that file actually exists afterward.
	if err := ensureJigToml(opts.TargetDir, opts.DryRun, out); err != nil {
		return err
	}

	filesToUpdate := []struct {
		TemplatePath string
		DestPath     string
	}{
		{TemplatePath: "module/Gemfile", DestPath: "Gemfile"},
		{TemplatePath: "module/Rakefile", DestPath: "Rakefile"},
		{TemplatePath: "module/spec/spec_helper.rb", DestPath: filepath.Join("spec", "spec_helper.rb")},
	}

	renderer := template.NewRenderer()

	for _, f := range filesToUpdate {
		fullDestPath := filepath.Join(opts.TargetDir, f.DestPath)

		content, err := renderer.Render(f.TemplatePath, nil)
		if err != nil {
			return fmt.Errorf("error while parsing template %s: %w", f.TemplatePath, err)
		}

		if opts.DryRun {
			fmt.Fprintf(out, "would write %s\n", fullDestPath)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fullDestPath), 0755); err != nil {
			return fmt.Errorf("error creating directory for %s: %w", fullDestPath, err)
		}

		if err := os.WriteFile(fullDestPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("error writing file %s: %w", fullDestPath, err)
		}
	}

	if err := removeStaleLockFiles(opts.TargetDir, opts.DryRun, out); err != nil {
		return err
	}
	warnAboutPDKArtifacts(opts.TargetDir, out)

	return nil
}

// removeStaleLockFiles deletes lockfiles left over from the Gemfile convert
// just replaced -- they lock a dependency set that no longer matches it, so
// bundler would fail resolving against them instead of just regenerating the
// lock as it should.
func removeStaleLockFiles(targetDir string, dryRun bool, out io.Writer) error {
	for _, name := range staleLockFiles {
		path := filepath.Join(targetDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return fmt.Errorf("failed to stat %s: %w", path, err)
		}

		if dryRun {
			fmt.Fprintf(out, "would remove %s\n", path)
			continue
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
		fmt.Fprintf(out, "removed %s\n", path)
	}
	return nil
}

// warnAboutPDKArtifacts reports any PDK-era files still sitting in the
// module. jig leaves them in place rather than guessing whether they still
// hold customization worth keeping (.rubocop.yml, .puppet-lint.rc), but
// they're no longer used by anything jig or voxbox does.
func warnAboutPDKArtifacts(targetDir string, out io.Writer) {
	var found []string
	for _, name := range pdkArtifacts {
		if _, err := os.Stat(filepath.Join(targetDir, name)); err == nil {
			found = append(found, name)
		}
	}
	if len(found) > 0 {
		fmt.Fprintf(out, "warning: PDK-era files no longer used by jig/voxbox, safe to remove: %s\n", strings.Join(found, ", "))
	}
}

// ensureJigToml writes jig.toml with its default (empty) content if the
// module doesn't already have one.
func ensureJigToml(targetDir string, dryRun bool, out io.Writer) error {
	jigTomlPath := filepath.Join(targetDir, config.ModuleConfigFileName)
	if _, err := os.Stat(jigTomlPath); !os.IsNotExist(err) {
		return err // nil when it already exists; a real stat error otherwise
	}

	if dryRun {
		fmt.Fprintf(out, "would create %s\n", jigTomlPath)
		return nil
	}
	if err := (config.ModuleConfig{}).Write(targetDir); err != nil {
		return err
	}
	fmt.Fprintf(out, "created %s\n", jigTomlPath)
	return nil
}

// createMetadata builds metadata.json from opts (filling in the module name
// from the target directory when it wasn't already resolved from a
// Modulefile) and writes it.
func createMetadata(opts ConvertOptions, metadataPath string, out io.Writer) error {
	name := opts.Name
	if name == "" {
		_, name = SplitForgeName(filepath.Base(opts.TargetDir))
	}

	meta := module.NewMetadata(name, opts.ForgeUser, opts.Author)
	if opts.License != "" {
		meta.License = opts.License
	}
	meta.Summary = opts.Summary
	meta.Source = opts.Source
	meta.ProjectPage = opts.ProjectPage
	if opts.Version != "" {
		meta.Version = opts.Version
	}
	if len(opts.Dependencies) > 0 {
		meta.Dependencies = opts.Dependencies
	}

	// A freshly created metadata.json can still fail validation -- a missing
	// -S/--source, or a Modulefile with a non-semver version -- so warn about
	// it now rather than leaving that for the next `jig build` to discover.
	printValidationWarnings(out, meta.Validate())

	if opts.DryRun {
		fmt.Fprintf(out, "would create %s\n", metadataPath)
	} else {
		if err := meta.Write(metadataPath); err != nil {
			return fmt.Errorf("failed to write metadata.json: %w", err)
		}
		fmt.Fprintf(out, "created %s\n", metadataPath)
	}

	if opts.HasModulefile {
		fmt.Fprintf(out, "warning: Modulefile is no longer used; its contents were carried into metadata.json. You can delete it.\n")
	}

	return nil
}

// printValidationWarnings reports every module.ValidationResult at its own
// severity (Severity implements Stringer), rather than flattening
// info/warning/error together under one label.
func printValidationWarnings(out io.Writer, results []module.ValidationResult) {
	for _, r := range results {
		fmt.Fprintf(out, "%s: %s - %s\n", r.Level, r.Field, r.Message)
	}
}

// repairMetadata reads an existing metadata.json. A JSON parse error is
// fatal -- jig won't guess at a broken file. Otherwise it modernizes the
// pre-2014 bare-string-list form of operatingsystem_support if present,
// fills in the keys that have a safe default (version, and the
// dependencies/requirements/operatingsystem_support/tags lists) when
// they're missing, warns about validation failures it can't fix on its own,
// and writes the file back only if something actually changed, so running
// convert twice is a no-op.
//
// Repair round-trips through an orderedObject rather than the Metadata
// struct, so keys jig doesn't model (a PDK-era "pdk-version" or
// "data_provider", say) survive untouched instead of being silently dropped
// on write, and the rewrite doesn't reorder every key the way marshaling a
// plain map would.
func repairMetadata(metadataPath string, dryRun bool, out io.Writer) error {
	content, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("%s: %w", metadataPath, err)
	}

	raw, err := newOrderedObject(content)
	if err != nil {
		return fmt.Errorf("%s: %w", metadataPath, err)
	}

	changed := false

	modernizedOS, err := modernizeOperatingSystemSupport(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", metadataPath, err)
	}
	if modernizedOS {
		fmt.Fprintf(out, "warning: operatingsystem_support used the pre-2014 bare-name list format, modernized to the current object format\n")
		changed = true
	}

	coerced, err := raw.MarshalJSON()
	if err != nil {
		return fmt.Errorf("%s: %w", metadataPath, err)
	}

	var meta module.Metadata
	if err := json.Unmarshal(coerced, &meta); err != nil {
		return fmt.Errorf("%s: %w", metadataPath, err)
	}

	setDefault := func(key string, value any) {
		encoded, _ := marshalNoEscape(value)
		raw.Set(key, encoded)
		changed = true
	}

	if meta.Version == "" {
		meta.Version = "0.1.0"
		setDefault("version", meta.Version)
		fmt.Fprintf(out, "warning: version missing from metadata.json, defaulting to 0.1.0\n")
	}
	if meta.Dependencies == nil {
		meta.Dependencies = []module.Dependency{}
		setDefault("dependencies", meta.Dependencies)
	}
	if meta.Requirements == nil {
		meta.Requirements = []module.Requirement{}
		setDefault("requirements", meta.Requirements)
	}
	if meta.OperatingSystem == nil {
		meta.OperatingSystem = []module.OperatingSystem{}
		setDefault("operatingsystem_support", meta.OperatingSystem)
	}
	if meta.Tags == nil {
		meta.Tags = []string{}
		setDefault("tags", meta.Tags)
	}

	printValidationWarnings(out, meta.Validate())

	if !changed {
		return nil
	}

	if dryRun {
		fmt.Fprintf(out, "would repair %s\n", metadataPath)
		return nil
	}

	// json.MarshalIndent would re-apply HTML escaping to raw's already-decoded
	// bytes, mangling a preserved version_requirement like ">= 3.0.0" into
	// its >-escaped form; an Encoder with SetEscapeHTML(false) does not.
	var pretty bytes.Buffer
	enc := json.NewEncoder(&pretty)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(raw); err != nil {
		return fmt.Errorf("failed to write repaired metadata.json: %w", err)
	}
	if err := os.WriteFile(metadataPath, pretty.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write repaired metadata.json: %w", err)
	}
	fmt.Fprintf(out, "repaired %s\n", metadataPath)
	return nil
}

// modernizeOperatingSystemSupport rewrites the pre-2014 form of
// operatingsystem_support -- a bare list of OS name strings, e.g.
// ["Debian", "RedHat"] -- into the object list module.OperatingSystem
// expects. Without this, a metadata.json from that era fails json.Unmarshal
// with a raw Go type-mismatch error instead of being repaired. Entries that
// are already objects (or anything jig doesn't recognize) are left alone;
// reports whether anything changed.
func modernizeOperatingSystemSupport(raw *orderedObject) (bool, error) {
	value, ok := raw.Get("operatingsystem_support")
	if !ok {
		return false, nil
	}

	var entries []json.RawMessage
	if err := json.Unmarshal(value, &entries); err != nil {
		// Not a list this function understands; leave it for the normal
		// unmarshal into Metadata to report.
		return false, nil
	}

	changed := false
	for i, entry := range entries {
		var name string
		if err := json.Unmarshal(entry, &name); err != nil {
			continue // already an object, or something else entirely
		}
		modernized, err := marshalNoEscape(module.OperatingSystem{Name: name, Release: []string{}})
		if err != nil {
			return false, err
		}
		entries[i] = modernized
		changed = true
	}
	if !changed {
		return false, nil
	}

	newValue, err := marshalNoEscape(entries)
	if err != nil {
		return false, err
	}
	raw.Set("operatingsystem_support", newValue)
	return true, nil
}
