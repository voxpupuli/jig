// SPDX-License-Identifier: GPL-3.0-or-later
package module

import (
	"net/url"
	"regexp"
)

type Severity int

const (
	Info Severity = iota
	Warning
	Error
)

func (s Severity) String() string {
	switch s {
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

type ValidationResult struct {
	Level   Severity
	Field   string
	Message string
}

func (m Metadata) Validate() []ValidationResult {
	var results []ValidationResult

	//
	// Name validation
	//
	if m.Name == "" {
		results = append(results, ValidationResult{
			Level:   Error,
			Field:   "name",
			Message: "name is required",
		})
	}

	validNameRe := regexp.MustCompile(`^[a-z][a-z0-9_]*-[a-z][a-z0-9_]*$`)
	if !validNameRe.MatchString(m.Name) {
		results = append(results, ValidationResult{
			Level:   Warning,
			Field:   "name",
			Message: "name must start with a lowercase letter and contain only lowercase letters, numbers, and underscores",
		})
	}

	if m.Version == "" {
		results = append(results, ValidationResult{
			Level:   Error,
			Field:   "version",
			Message: "version is required",
		})
	}

	validVersionRe := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if m.Version != "" && !validVersionRe.MatchString(m.Version) {
		results = append(results, ValidationResult{
			Level:   Error,
			Field:   "version",
			Message: "version must be a valid semver string (MAJOR.MINOR.PATCH)",
		})
	}

	if m.Author == "" {
		results = append(results, ValidationResult{
			Level:   Error,
			Field:   "author",
			Message: "author is required",
		})
	}

	if m.License == "" {
		results = append(results, ValidationResult{
			Level:   Error,
			Field:   "license",
			Message: "license is required",
		})
	}

	if m.Summary == "" {
		results = append(results, ValidationResult{
			Level:   Error,
			Field:   "summary",
			Message: "summary is required",
		})
	}

	if m.Source == "" {
		results = append(results, ValidationResult{
			Level:   Error,
			Field:   "source",
			Message: "source is required",
		})
	}

	if m.Source != "" && !isValidURL(m.Source) {
		results = append(results, ValidationResult{
			Level:   Error,
			Field:   "source",
			Message: "source must be a valid URL",
		})
	}

	if m.ProjectPage != "" && !isValidURL(m.ProjectPage) {
		results = append(results, ValidationResult{
			Level:   Error,
			Field:   "project_page",
			Message: "project_page must be a valid URL",
		})
	}

	if m.IssuesURL != "" && !isValidURL(m.IssuesURL) {
		results = append(results, ValidationResult{
			Level:   Error,
			Field:   "issues_url",
			Message: "issues_url must be a valid URL",
		})
	}

	if m.HasTemplateSettings() {
		results = append(results, ValidationResult{
			Level:   Warning,
			Field:   "template-url",
			Message: "template settings in metadata.json are not supported; move template-url/template-ref/template-commit to the [template] section of jig.toml and remove them from metadata.json",
		})
	}

	return results
}

func isValidURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
