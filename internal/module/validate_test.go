// SPDX-License-Identifier: GPL-3.0-or-later
package module

import (
	"testing"
)

func TestSeverityString(t *testing.T) {
	cases := []struct {
		severity Severity
		expected string
	}{
		{Info, "info"},
		{Warning, "warning"},
		{Error, "error"},
		{Severity(99), "unknown"},
	}

	for _, tc := range cases {
		if got := tc.severity.String(); got != tc.expected {
			t.Errorf("Severity(%d).String(): got %q, want %q", tc.severity, got, tc.expected)
		}
	}
}

// validMetadata returns a Metadata that passes all validations, used as a
// baseline that individual test cases can modify.
func validMetadata() Metadata {
	return Metadata{
		Name:    "author-module",
		Version: "0.1.0",
		Author:  "Author Name",
		License: "Apache-2.0",
		Summary: "A test module",
		Source:  "https://github.com/author/module",
	}
}

// findResults filters validation results by field and severity.
func findResults(results []ValidationResult, field string, level Severity) []ValidationResult {
	var found []ValidationResult
	for _, r := range results {
		if r.Field == field && r.Level == level {
			found = append(found, r)
		}
	}
	return found
}

func TestValidate_ValidMetadata(t *testing.T) {
	results := validMetadata().Validate()
	if len(results) != 0 {
		t.Errorf("expected no validation results for valid metadata, got %d: %v", len(results), results)
	}
}

func TestValidate_RequiredFields(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Metadata)
		field  string
		level  Severity
	}{
		{"empty version", func(m *Metadata) { m.Version = "" }, "version", Error},
		{"empty author", func(m *Metadata) { m.Author = "" }, "author", Error},
		{"empty license", func(m *Metadata) { m.License = "" }, "license", Error},
		{"empty summary", func(m *Metadata) { m.Summary = "" }, "summary", Error},
		{"empty source", func(m *Metadata) { m.Source = "" }, "source", Error},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validMetadata()
			tc.mutate(&m)
			results := m.Validate()
			if len(findResults(results, tc.field, tc.level)) == 0 {
				t.Errorf("expected %s-level result for field %q, got none. all results: %v", tc.level, tc.field, results)
			}
		})
	}
}

func TestValidate_NameFormat(t *testing.T) {
	cases := []struct {
		name        string
		moduleName  string
		wantError   bool
		wantWarning bool
	}{
		{"valid", "author-module", false, false},
		{"valid with numbers", "author1-module2", false, false},
		{"valid with underscores", "my_author-my_module", false, false},
		{"empty name", "", true, true}, // triggers both error and warning
		{"missing delimiter", "authormodule", false, true},
		{"uppercase author", "Author-module", false, true},
		{"uppercase module", "author-Module", false, true},
		{"starts with number", "1author-module", false, true},
		{"trailing dash", "author-module-", false, true},
		{"leading dash", "-author-module", false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validMetadata()
			m.Name = tc.moduleName

			results := m.Validate()

			errors := findResults(results, "name", Error)
			warnings := findResults(results, "name", Warning)

			if tc.wantError && len(errors) == 0 {
				t.Errorf("expected name Error, got none. all results: %v", results)
			}
			if !tc.wantError && len(errors) > 0 {
				t.Errorf("unexpected name Error: %v", errors)
			}
			if tc.wantWarning && len(warnings) == 0 {
				t.Errorf("expected name Warning, got none. all results: %v", results)
			}
			if !tc.wantWarning && len(warnings) > 0 {
				t.Errorf("unexpected name Warning: %v", warnings)
			}
		})
	}
}

func TestValidate_VersionFormat(t *testing.T) {
	cases := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"valid semver", "1.0.0", false},
		{"major minor patch", "10.20.30", false},
		{"not semver", "not-a-version", true},
		{"partial semver", "1.0", true},
		{"too many parts", "1.0.0.0", true},
		{"leading v", "v1.0.0", true},
		{"empty", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validMetadata()
			m.Version = tc.version
			results := m.Validate()
			errors := findResults(results, "version", Error)
			if tc.wantErr && len(errors) == 0 {
				t.Errorf("expected version Error for %q, got none", tc.version)
			}
			if !tc.wantErr && len(errors) > 0 {
				t.Errorf("unexpected version Error for %q: %v", tc.version, errors)
			}
		})
	}
}

func TestValidate_NameEdgeCases(t *testing.T) {
	cases := []struct {
		name        string
		moduleName  string
		wantWarning bool
	}{
		{"name with spaces", "author-my module", true},
		{"name with slash", "author/module", true},
		{"name with dot", "author.name-module", true},
		{"unicode characters", "authör-module", true},
		{"only delimiter", "-", true},
		{"whitespace only", "   ", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validMetadata()
			m.Name = tc.moduleName
			results := m.Validate()
			warnings := findResults(results, "name", Warning)
			if tc.wantWarning && len(warnings) == 0 {
				t.Errorf("expected name Warning for %q, got none", tc.moduleName)
			}
		})
	}
}

func TestValidate_URLFields(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*Metadata)
		field     string
		wantError bool
	}{
		{"valid source URL", func(m *Metadata) { m.Source = "https://github.com/author/module" }, "source", false},
		{"source http scheme", func(m *Metadata) { m.Source = "http://example.com" }, "source", false},
		{"source not a URL", func(m *Metadata) { m.Source = "not-a-url" }, "source", true},
		{"source missing scheme", func(m *Metadata) { m.Source = "github.com/author/module" }, "source", true},
		{"source ftp scheme", func(m *Metadata) { m.Source = "ftp://example.com" }, "source", true},
		{"valid project_page", func(m *Metadata) { m.ProjectPage = "https://example.com" }, "project_page", false},
		{"project_page not a URL", func(m *Metadata) { m.ProjectPage = "not-a-url" }, "project_page", true},
		{"project_page empty is ok", func(m *Metadata) { m.ProjectPage = "" }, "project_page", false},
		{"valid issues_url", func(m *Metadata) { m.IssuesURL = "https://github.com/author/module/issues" }, "issues_url", false},
		{"issues_url not a URL", func(m *Metadata) { m.IssuesURL = "not-a-url" }, "issues_url", true},
		{"issues_url empty is ok", func(m *Metadata) { m.IssuesURL = "" }, "issues_url", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validMetadata()
			tc.mutate(&m)
			results := m.Validate()
			errors := findResults(results, tc.field, Error)
			if tc.wantError && len(errors) == 0 {
				t.Errorf("expected Error for field %q, got none. all results: %v", tc.field, results)
			}
			if !tc.wantError && len(errors) > 0 {
				t.Errorf("unexpected Error for field %q: %v", tc.field, errors)
			}
		})
	}
}

// Template keys in metadata.json (written by jig 1.x) are not supported;
// their presence must produce a warning pointing at jig.toml, and clean
// metadata must not.
func TestValidate_TemplateSettingsWarn(t *testing.T) {
	cases := []struct {
		name   string
		modify func(*Metadata)
		warns  bool
	}{
		{"no template keys", func(m *Metadata) {}, false},
		{"template-url", func(m *Metadata) { m.TemplateURL = "ssh://git@example.com/t.git" }, true},
		{"template-ref only", func(m *Metadata) { m.TemplateRef = "main" }, true},
		{"template-commit only", func(m *Metadata) { m.TemplateCommit = "abc123" }, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validMetadata()
			tc.modify(&m)
			warnings := findResults(m.Validate(), "template-url", Warning)
			if tc.warns && len(warnings) == 0 {
				t.Error("expected a warning about template settings in metadata.json")
			}
			if !tc.warns && len(warnings) != 0 {
				t.Errorf("expected no template warning, got %v", warnings)
			}
		})
	}
}
