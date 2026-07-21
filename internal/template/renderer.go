// SPDX-License-Identifier: GPL-3.0-or-later
package template

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	tmplpkg "text/template"
)

//go:embed all:templates
var embeddedTemplates embed.FS

// TmplSuffix marks a template file as rendered: "README.md.tmpl" is rendered
// through text/template and written as "README.md". Files without the suffix
// are copied verbatim.
const TmplSuffix = ".tmpl"

type Renderer struct {
	externalDir string
}

func NewRenderer() *Renderer {
	return &Renderer{}
}

func NewRendererWithExternalDir(dir string) *Renderer {
	return &Renderer{
		externalDir: dir,
	}
}

// validateName rejects names that would escape the template directory and
// returns the cleaned slash-separated logical name.
func validateName(templateName string) (string, error) {
	if templateName == "" {
		return "", fmt.Errorf("template name cannot be empty")
	}

	// Prevent path traversal when reading from the external directory.
	// We check the embedded path too for consistency -- embed.FS would
	// reject ".." internally, but rejecting it here gives a clearer error.
	cleaned := filepath.Clean(templateName)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("invalid template name %q: must be a relative path within the template directory", templateName)
	}

	// embed.FS always uses forward slashes on every OS. Template names are
	// logical slash-paths, but on Windows filepath.Clean rewrites them with
	// backslashes (and a Windows caller may pass backslashes directly), so
	// convert any backslashes before building the embedded lookup path.
	return strings.ReplaceAll(cleaned, `\`, "/"), nil
}

// resolve looks up a logical template name and returns its content and
// whether it is a template (a .tmpl file, to be rendered) or a verbatim file
// (to be copied as-is). The external directory takes precedence over the
// embedded templates; within one source, having both "name" and "name.tmpl"
// is an error because both would produce the same output file.
func (r Renderer) resolve(name string) ([]byte, bool, error) {
	if r.externalDir != "" {
		extBase := filepath.Clean(r.externalDir)
		extPath := filepath.Join(extBase, filepath.FromSlash(name))
		// Double-check the joined path stays within the external dir.
		if !strings.HasPrefix(extPath, extBase+string(filepath.Separator)) {
			return nil, false, fmt.Errorf("invalid template name %q: resolves outside template directory", name)
		}

		content, isTemplate, found, err := readOne(
			func(p string) ([]byte, error) { return os.ReadFile(p) },
			extPath+TmplSuffix, extPath, name,
		)
		if err != nil {
			return nil, false, err
		}
		if found {
			return content, isTemplate, nil
		}
		// not found in external dir, fall through to embedded templates
	}

	embPath := "templates/" + name
	content, isTemplate, found, err := readOne(
		func(p string) ([]byte, error) { return embeddedTemplates.ReadFile(p) },
		embPath+TmplSuffix, embPath, name,
	)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, fmt.Errorf("template %s not found", name)
	}
	return content, isTemplate, nil
}

// readOne reads a logical template from a single source, preferring the
// .tmpl variant. It errors if both variants exist, since they would render
// to the same destination.
func readOne(read func(string) ([]byte, error), tmplPath, plainPath, name string) (content []byte, isTemplate, found bool, err error) {
	tmplContent, tmplErr := read(tmplPath)
	plainContent, plainErr := read(plainPath)

	if tmplErr != nil && !isNotFound(tmplErr) {
		return nil, false, false, fmt.Errorf("failed to read template %s: %w", name+TmplSuffix, tmplErr)
	}
	if plainErr != nil && !isNotFound(plainErr) {
		return nil, false, false, fmt.Errorf("failed to read template %s: %w", name, plainErr)
	}

	switch {
	case tmplErr == nil && plainErr == nil:
		return nil, false, false, fmt.Errorf("both %s and %s%s exist and would produce the same file; remove one", name, name, TmplSuffix)
	case tmplErr == nil:
		return tmplContent, true, true, nil
	case plainErr == nil:
		return plainContent, false, true, nil
	default:
		return nil, false, false, nil
	}
}

func isNotFound(err error) bool {
	// os.ReadFile on a directory returns EISDIR-style errors, not IsNotExist;
	// treat only genuine absence as "not found" so real read failures surface.
	return os.IsNotExist(err)
}

// Render resolves a logical template name and returns its content. Names
// never include the .tmpl suffix: Render("module/README.md") renders
// "module/README.md.tmpl" if it exists, otherwise returns the verbatim
// content of "module/README.md".
func (r Renderer) Render(templateName string, data any) (string, error) {
	name, err := validateName(templateName)
	if err != nil {
		return "", err
	}

	content, isTemplate, err := r.resolve(name)
	if err != nil {
		return "", err
	}

	if !isTemplate {
		return string(content), nil
	}

	funcMap := tmplpkg.FuncMap{
		"upperFirst": func(s string) string {
			if s == "" {
				return ""
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
		"pascalCase": func(s string) string {
			parts := strings.Split(s, "_")
			for i, p := range parts {
				if p != "" {
					parts[i] = strings.ToUpper(p[:1]) + p[1:]
				}
			}
			return strings.Join(parts, "")
		},
	}

	t, err := tmplpkg.New(templateName).Funcs(funcMap).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to render template %s: %w", templateName, err)
	}
	return buf.String(), nil
}

// ListTree returns the sorted logical names of every file under the named
// top-level template directory (e.g. "module"), unioned across the embedded
// templates and the external directory. Logical names are slash-separated,
// relative to root, with the .tmpl suffix stripped -- each one is both a
// valid Render name (prefixed with root) and the file's destination path
// relative to the output directory. Within a single source, a file and its
// .tmpl variant mapping to the same logical name is an error.
func (r Renderer) ListTree(root string) ([]string, error) {
	rootName, err := validateName(root)
	if err != nil {
		return nil, err
	}

	// Collision detection (foo alongside foo.tmpl) applies within a single
	// source; across sources the external tree simply overrides the embedded
	// one, so each source gets its own seen-map before the union.
	embedded := map[string]bool{}
	embRoot := "templates/" + rootName
	err = fs.WalkDir(embeddedTemplates, embRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		return addLogical(embedded, strings.TrimPrefix(path, embRoot+"/"), "embedded templates")
	})
	if err != nil {
		return nil, err
	}

	external := map[string]bool{}
	if r.externalDir != "" {
		extRoot := filepath.Join(filepath.Clean(r.externalDir), filepath.FromSlash(rootName))
		if info, statErr := os.Stat(extRoot); statErr == nil && info.IsDir() {
			err = filepath.WalkDir(extRoot, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(extRoot, path)
				if err != nil {
					return err
				}
				return addLogical(external, filepath.ToSlash(rel), r.externalDir)
			})
			if err != nil {
				return nil, err
			}
		}
	}

	names := make([]string, 0, len(embedded)+len(external))
	for name := range embedded {
		names = append(names, name)
	}
	for name := range external {
		if !embedded[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func addLogical(seen map[string]bool, rel string, source string) error {
	name := strings.TrimSuffix(rel, TmplSuffix)
	if name == "" {
		return fmt.Errorf("invalid template file name %q in %s", rel, source)
	}
	if seen[name] {
		return fmt.Errorf("both %s and %s%s exist in %s and would produce the same file; remove one", name, name, TmplSuffix, source)
	}
	seen[name] = true
	return nil
}

func DumpTemplates(dest string) error {
	return fs.WalkDir(embeddedTemplates, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Strip the "templates/" prefix to get the relative path
		relPath := strings.TrimPrefix(path, "templates/")
		if relPath == "" {
			return nil
		}

		destPath := filepath.Join(dest, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		content, err := embeddedTemplates.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded template %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
		}

		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", destPath, err)
		}

		fmt.Printf("wrote %s\n", destPath)
		return nil
	})
}
