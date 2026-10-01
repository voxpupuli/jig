// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/voxpupuli/jig/v2/internal/config"
	"github.com/voxpupuli/jig/v2/internal/template"
)

// varsTemplateDir writes a template dir declaring two variables, with a
// README that shows them.
func varsTemplateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		template.ManifestFileName: `
[vars.ci]
type    = "string"
default = "github"

[vars.junit]
type    = "bool"
default = false
`,
		"module/README.md.tmpl": "ci={{.Vars.ci}} junit={{.Vars.junit}}\n",
	}
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func loadVars(t *testing.T, dir string) map[string]any {
	t.Helper()
	cfg, err := config.LoadModuleConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Template.Vars
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// jig new module resolves flag > user config (module, author, default) >
// template default, and records every resolved value in jig.toml.
func TestNewModuleCmd_ResolvesAndRecordsVars(t *testing.T) {
	tmplDir := varsTemplateDir(t)
	work := t.TempDir()
	t.Chdir(work)

	a := testApp(config.Config{TemplateVars: map[string]any{
		"default": map[string]any{"ci": "jenkins", "junit": true},
		"acme":    map[string]any{"widget": map[string]any{"ci": "gitlab"}},
	}})
	parent := a.newCmd()
	parent.SetArgs([]string{"module", "--skip-interview", "-u", "acme", "-a", "Ada", "--template-dir", tmplDir, "widget"})
	if err := parent.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	moduleDir := filepath.Join(work, "widget")
	want := map[string]any{"ci": "gitlab", "junit": true}
	if got := loadVars(t, moduleDir); !reflect.DeepEqual(got, want) {
		t.Errorf("[template.vars]: got %v, want %v", got, want)
	}
	if got := readFile(t, filepath.Join(moduleDir, "README.md")); got != "ci=gitlab junit=true\n" {
		t.Errorf("README.md: got %q", got)
	}
}

func TestNewModuleCmd_FlagBeatsUserConfig(t *testing.T) {
	tmplDir := varsTemplateDir(t)
	work := t.TempDir()
	t.Chdir(work)

	a := testApp(config.Config{TemplateVars: map[string]any{
		"default": map[string]any{"ci": "jenkins"},
	}})
	parent := a.newCmd()
	parent.SetArgs([]string{"module", "--skip-interview", "-u", "acme", "-a", "Ada", "--template-dir", tmplDir, "--template-var", "ci=drone", "widget"})
	if err := parent.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"ci": "drone", "junit": false}
	if got := loadVars(t, filepath.Join(work, "widget")); !reflect.DeepEqual(got, want) {
		t.Errorf("[template.vars]: got %v, want %v", got, want)
	}
}

// A --template-var the template does not declare must stop jig new module
// before anything is written.
func TestNewModuleCmd_UndeclaredVarIsError(t *testing.T) {
	tmplDir := varsTemplateDir(t)
	work := t.TempDir()
	t.Chdir(work)

	parent := testApp(config.Config{}).newCmd()
	parent.SetArgs([]string{"module", "--skip-interview", "-u", "acme", "-a", "Ada", "--template-dir", tmplDir, "--template-var", "jnuit=true", "widget"})
	parent.SetOut(&strings.Builder{})
	parent.SetErr(&strings.Builder{})
	err := parent.Execute()
	if err == nil || !strings.Contains(err.Error(), "jnuit") {
		t.Fatalf("expected an error naming the undeclared variable, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(work, "widget")); !os.IsNotExist(statErr) {
		t.Error("no module directory must be created when a --template-var is invalid")
	}
}

// renew renders with jig.toml's values and ignores the user config, and a
// --template-var both overrides and is saved to jig.toml.
func TestRenewCmd_TemplateVars(t *testing.T) {
	tmplDir := varsTemplateDir(t)
	dir := renewModuleDir(t, "", []string{"README.md"})
	cfg, err := config.LoadModuleConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Template.Vars = map[string]any{"ci": "circle"}
	if err := cfg.Write(dir); err != nil {
		t.Fatal(err)
	}
	a := testApp(config.Config{TemplateVars: map[string]any{"default": map[string]any{"junit": true}}})

	if out, err := runRenew(t, a, "--template-dir", tmplDir); err != nil {
		t.Fatalf("unexpected error: %v (output: %q)", err, out)
	}
	if got := readFile(t, filepath.Join(dir, "README.md")); got != "ci=circle junit=false\n" {
		t.Errorf("renew must use jig.toml and template defaults only, got %q", got)
	}

	out, err := runRenew(t, a, "--template-dir", tmplDir, "--template-var", "junit=true")
	if err != nil {
		t.Fatalf("unexpected error: %v (output: %q)", err, out)
	}
	if got := readFile(t, filepath.Join(dir, "README.md")); got != "ci=circle junit=true\n" {
		t.Errorf("README.md: got %q", got)
	}
	want := map[string]any{"ci": "circle", "junit": true}
	if got := loadVars(t, dir); !reflect.DeepEqual(got, want) {
		t.Errorf("[template.vars]: got %v, want %v", got, want)
	}
	if !strings.Contains(out, "updated [template.vars]") {
		t.Errorf("output should report the jig.toml update, got %q", out)
	}
}

func TestRenewCmd_DryRunDoesNotSaveVars(t *testing.T) {
	tmplDir := varsTemplateDir(t)
	dir := renewModuleDir(t, "", []string{"README.md"})

	if out, err := runRenew(t, testApp(config.Config{}), "--template-dir", tmplDir, "--template-var", "junit=true", "--dry-run"); err != nil {
		t.Fatalf("unexpected error: %v (output: %q)", err, out)
	}
	if got := loadVars(t, dir); got != nil {
		t.Errorf("dry run must not save variables, got %v", got)
	}
}

// convert takes the template source flags and records the source and
// resolved variables in jig.toml.
func TestConvertCmd_RecordsTemplateVars(t *testing.T) {
	tmplDir := varsTemplateDir(t)
	moduleDir := filepath.Join(t.TempDir(), "acme-widget")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(moduleDir)

	a := testApp(config.Config{
		ForgeUsername: "acme",
		Author:        "Ada",
		TemplateVars: map[string]any{
			"acme": map[string]any{"junit": true},
		},
	})
	out, err := runConvert(t, a, "--skip-interview", "-S", "https://example.com/widget", "--template-dir", tmplDir, "--template-var", "ci=gitlab")
	if err != nil {
		t.Fatalf("unexpected error: %v (output: %q)", err, out)
	}

	want := map[string]any{"ci": "gitlab", "junit": true}
	if got := loadVars(t, moduleDir); !reflect.DeepEqual(got, want) {
		t.Errorf("[template.vars]: got %v, want %v", got, want)
	}
}

// A template url recorded in jig.toml must not be fetched by convert: with
// an unreachable url, convert still succeeds offline as it does on main.
func TestConvertCmd_DoesNotFetchRecordedURL(t *testing.T) {
	moduleDir := filepath.Join(t.TempDir(), "acme-widget")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(moduleDir)
	jigToml := "[template]\nurl = \"" + filepath.Join(t.TempDir(), "no-such-repo") + "\"\ncommit = \"abc\"\n"
	if err := os.WriteFile(filepath.Join(moduleDir, config.ModuleConfigFileName), []byte(jigToml), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runConvert(t, testApp(config.Config{ForgeUsername: "acme", Author: "Ada"}), "--skip-interview", "-S", "https://example.com/widget")
	if err != nil {
		t.Fatalf("convert must not need the recorded template url, got: %v (output: %q)", err, out)
	}
	if got := readFile(t, filepath.Join(moduleDir, config.ModuleConfigFileName)); got != jigToml {
		t.Errorf("jig.toml must be untouched, got %q", got)
	}
}

// A broken [template.vars] in the user config must be reported before the
// interview, not after the user has answered it.
func TestNewModuleCmd_InvalidUserConfigFailsEarly(t *testing.T) {
	tmplDir := varsTemplateDir(t)
	t.Chdir(t.TempDir())

	parent := testApp(config.Config{TemplateVars: map[string]any{"default": map[string]any{"camelCase": true}}}).newCmd()
	// No --skip-interview: reaching the interview would block on stdin.
	parent.SetArgs([]string{"module", "-u", "acme", "-a", "Ada", "--template-dir", tmplDir, "widget"})
	parent.SetOut(&strings.Builder{})
	parent.SetErr(&strings.Builder{})
	if err := parent.Execute(); err == nil || !strings.Contains(err.Error(), "camelCase") {
		t.Fatalf("expected an error naming the bad key, got: %v", err)
	}
}

// renew --template-var with an unchanged list value must not rewrite
// jig.toml.
func TestRenewCmd_UnchangedListVarDoesNotRewrite(t *testing.T) {
	tmplDir := t.TempDir()
	manifest := "[vars.fixtures]\ntype = \"list\"\n"
	if err := os.WriteFile(filepath.Join(tmplDir, template.ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmplDir, "module"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmplDir, "module", "README.md.tmpl"), []byte("{{ .Vars.fixtures }}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := renewModuleDir(t, "", []string{"README.md"})
	path := filepath.Join(dir, config.ModuleConfigFileName)
	jigToml := "# keep me\n[template.vars]\nfixtures = ['a/b']\n[renew]\npaths = ['README.md']\n"
	if err := os.WriteFile(path, []byte(jigToml), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runRenew(t, testApp(config.Config{}), "--template-dir", tmplDir, "--template-var", "fixtures=a/b")
	if err != nil {
		t.Fatalf("unexpected error: %v (output: %q)", err, out)
	}
	if got := readFile(t, path); got != jigToml {
		t.Errorf("jig.toml must not be rewritten for an unchanged value, got %q", got)
	}
	if strings.Contains(out, "updated [template.vars]") {
		t.Errorf("output must not report an update, got %q", out)
	}
}

func TestRunVarInterview(t *testing.T) {
	manifest := template.Manifest{Vars: map[string]template.VarSpec{
		"junit":    {Type: template.VarTypeBool, Prompt: true, Description: "Enable JUnit"},
		"fixtures": {Type: template.VarTypeList, Prompt: true},
		"ci":       {Type: template.VarTypeString, Prompt: true},
		"quiet":    {Type: template.VarTypeBool},
	}}
	vars := map[string]any{"junit": false, "fixtures": []string{"x/y"}, "ci": "github", "quiet": false}
	flagVars := map[string]any{"ci": "github"}

	answers := map[string][]string{
		"fixtures":             {""},              // empty answer keeps the default
		"Enable JUnit (junit)": {"maybe", "true"}, // invalid answer is asked again
	}
	var asked []string
	ask := func(question, defaultVal string) string {
		asked = append(asked, question+" ["+defaultVal+"]")
		queue := answers[question]
		if len(queue) == 0 {
			t.Fatalf("unexpected question %q", question)
		}
		answers[question] = queue[1:]
		if queue[0] == "" {
			return defaultVal
		}
		return queue[0]
	}

	runVarInterview(manifest, vars, flagVars, ask)

	wantAsked := []string{"fixtures [x/y]", "Enable JUnit (junit) [false]", "Enable JUnit (junit) [false]"}
	if !reflect.DeepEqual(asked, wantAsked) {
		t.Errorf("questions: got %q, want %q (ci is set by flag, quiet has no prompt)", asked, wantAsked)
	}
	want := map[string]any{"junit": true, "fixtures": []string{"x/y"}, "ci": "github", "quiet": false}
	if !reflect.DeepEqual(vars, want) {
		t.Errorf("vars: got %v, want %v", vars, want)
	}
}
