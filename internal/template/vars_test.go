// SPDX-License-Identifier: GPL-3.0-or-later
package template

import (
	"reflect"
	"strings"
	"testing"
)

func testManifest() Manifest {
	return Manifest{Vars: map[string]VarSpec{
		"enable_junit": {Type: VarTypeBool, Default: false},
		"ci":           {Type: VarTypeString, Default: "github"},
		"retries":      {Type: VarTypeInt, Default: int64(1)},
		"fixtures":     {Type: VarTypeList, Default: []string{}},
	}}
}

func TestValidateVarName(t *testing.T) {
	for _, name := range []string{"a", "enable_junit", "x2", "a_b_c"} {
		if err := ValidateVarName(name); err != nil {
			t.Errorf("%q should be valid, got: %v", name, err)
		}
	}
	for _, name := range []string{"", "enableJunit", "Enable", "_x", "2x", "a-b", "a.b", "a b"} {
		if err := ValidateVarName(name); err == nil {
			t.Errorf("%q should be rejected", name)
		}
	}
}

func TestValidateVarValue(t *testing.T) {
	valid := []any{"s", true, int64(1), 1, 1.5, []string{"a"}, []any{"a", int64(1)}}
	for _, v := range valid {
		if err := ValidateVarValue("x", v); err != nil {
			t.Errorf("%v (%T) should be valid, got: %v", v, v, err)
		}
	}
	invalid := []any{map[string]any{"a": 1}, []any{map[string]any{}}, []any{[]any{"nested"}}, nil}
	for _, v := range invalid {
		if err := ValidateVarValue("x", v); err == nil {
			t.Errorf("%v (%T) should be rejected", v, v)
		}
	}
}

func TestParseVarFlags(t *testing.T) {
	got, err := ParseVarFlags([]string{"a=1", "b=x=y", "l=one", "l=two", "e="})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string][]string{"a": {"1"}, "b": {"x=y"}, "l": {"one", "two"}, "e": {""}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for _, bad := range []string{"novalue", "Bad=1", "=1"} {
		if _, err := ParseVarFlags([]string{bad}); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestCoerceFlags(t *testing.T) {
	m := testManifest()
	got, err := m.CoerceFlags(map[string][]string{
		"enable_junit": {"true"},
		"ci":           {"gitlab"},
		"retries":      {"3"},
		"fixtures":     {"a/b", "c/d"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"enable_junit": true, "ci": "gitlab", "retries": int64(3), "fixtures": []string{"a/b", "c/d"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCoerceFlags_EmptyList(t *testing.T) {
	got, err := testManifest().CoerceFlags(map[string][]string{"fixtures": {""}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got["fixtures"], []string{}) {
		t.Errorf("an empty value must give an empty list, got %#v", got["fixtures"])
	}
}

// "false" must become a real boolean: as a string it would be truthy in a
// template's if.
func TestCoerceFlags_FalseIsFalse(t *testing.T) {
	got, err := testManifest().CoerceFlags(map[string][]string{"enable_junit": {"false"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["enable_junit"] != false {
		t.Errorf("got %#v, want false", got["enable_junit"])
	}
}

func TestCoerceFlags_Errors(t *testing.T) {
	cases := []struct {
		name  string
		flags map[string][]string
		want  string
	}{
		{"undeclared", map[string][]string{"enable_junt": {"true"}}, "does not declare"},
		{"bad bool", map[string][]string{"enable_junit": {"maybe"}}, "bool"},
		{"bad int", map[string][]string{"retries": {"three"}}, "int"},
		{"repeated scalar", map[string][]string{"ci": {"a", "b"}}, "one value"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := testManifest().CoerceFlags(tc.flags)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q, got: %v", tc.want, err)
			}
		})
	}
}

func TestResolveVars_Precedence(t *testing.T) {
	m := testManifest()
	cases := []struct {
		name   string
		flags  map[string]any
		layers VarLayers
		want   map[string]any
	}{
		{
			name: "declared defaults fill everything",
			want: map[string]any{"enable_junit": false, "ci": "github", "retries": int64(1), "fixtures": []string{}},
		},
		{
			name:   "global layers in order, most specific first",
			layers: VarLayers{Global: []map[string]any{{"ci": "gitlab"}, {"ci": "jenkins", "enable_junit": true}}},
			want:   map[string]any{"enable_junit": true, "ci": "gitlab", "retries": int64(1), "fixtures": []string{}},
		},
		{
			name: "module values beat global layers",
			layers: VarLayers{
				Module: map[string]any{"ci": "circle"},
				Global: []map[string]any{{"ci": "gitlab"}},
			},
			want: map[string]any{"enable_junit": false, "ci": "circle", "retries": int64(1), "fixtures": []string{}},
		},
		{
			name:   "flags beat everything",
			flags:  map[string]any{"ci": "drone"},
			layers: VarLayers{Module: map[string]any{"ci": "circle"}, Global: []map[string]any{{"ci": "gitlab"}}},
			want:   map[string]any{"enable_junit": false, "ci": "drone", "retries": int64(1), "fixtures": []string{}},
		},
		{
			name:   "stored values are normalized to declared types",
			layers: VarLayers{Module: map[string]any{"fixtures": []any{"a/b"}, "retries": 5}},
			want:   map[string]any{"enable_junit": false, "ci": "github", "retries": int64(5), "fixtures": []string{"a/b"}},
		},
		{
			name:   "undeclared module values pass through",
			layers: VarLayers{Module: map[string]any{"legacy": "kept"}},
			want:   map[string]any{"enable_junit": false, "ci": "github", "retries": int64(1), "fixtures": []string{}, "legacy": "kept"},
		},
		{
			name:   "undeclared global values are ignored",
			layers: VarLayers{Global: []map[string]any{{"other_template_var": true}}},
			want:   map[string]any{"enable_junit": false, "ci": "github", "retries": int64(1), "fixtures": []string{}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := m.ResolveVars(tc.flags, tc.layers)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveVars_TypeMismatch(t *testing.T) {
	m := testManifest()
	if _, err := m.ResolveVars(nil, VarLayers{Module: map[string]any{"enable_junit": "yes"}}); err == nil {
		t.Error("a module value of the wrong type must be an error")
	}
	_, err := m.ResolveVars(nil, VarLayers{Global: []map[string]any{{"retries": "many"}}})
	if err == nil || !strings.Contains(err.Error(), "user config") {
		t.Errorf("a global value of the wrong type must be an error naming the user config, got: %v", err)
	}
}

func TestParseVarInput(t *testing.T) {
	m := testManifest()
	got, err := m.ParseVarInput("fixtures", " a/b , c/d ,")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"a/b", "c/d"}) {
		t.Errorf("got %v", got)
	}
	if _, err := m.ParseVarInput("enable_junit", "perhaps"); err == nil {
		t.Error("an invalid bool answer must be an error")
	}
	if FormatVarValue([]string{"a/b", "c/d"}) != "a/b, c/d" {
		t.Error("lists must format comma-separated so they parse back")
	}
}
