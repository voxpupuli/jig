// SPDX-License-Identifier: GPL-3.0-or-later
package template

import (
	"fmt"
	"io"
	"maps"
	"os"
	"reflect"
	"sort"
	tmplpkg "text/template"
	"text/template/parse"
)

// varRefs returns the sorted, de-duplicated variable names a parsed template
// references as .Vars.<name> or $.Vars.<name>, including inside any
// templates it defines.
func varRefs(t *tmplpkg.Template) []string {
	seen := map[string]bool{}
	for _, sub := range t.Templates() {
		if sub.Tree != nil {
			collectVarRefs(sub.Tree.Root, seen)
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func collectVarRefs(node parse.Node, seen map[string]bool) {
	switch n := node.(type) {
	case nil:
	case *parse.ListNode:
		if n == nil {
			return
		}
		for _, child := range n.Nodes {
			collectVarRefs(child, seen)
		}
	case *parse.ActionNode:
		collectVarRefs(n.Pipe, seen)
	case *parse.IfNode:
		collectBranch(&n.BranchNode, seen)
	case *parse.RangeNode:
		collectBranch(&n.BranchNode, seen)
	case *parse.WithNode:
		collectBranch(&n.BranchNode, seen)
	case *parse.TemplateNode:
		collectVarRefs(n.Pipe, seen)
	case *parse.PipeNode:
		if n == nil {
			return
		}
		for _, cmd := range n.Cmds {
			collectVarRefs(cmd, seen)
		}
	case *parse.CommandNode:
		for _, arg := range n.Args {
			collectVarRefs(arg, seen)
		}
	case *parse.ChainNode:
		collectVarRefs(n.Node, seen)
	case *parse.FieldNode:
		if len(n.Ident) >= 2 && n.Ident[0] == "Vars" {
			seen[n.Ident[1]] = true
		}
	case *parse.VariableNode:
		if len(n.Ident) >= 3 && n.Ident[0] == "$" && n.Ident[1] == "Vars" {
			seen[n.Ident[2]] = true
		}
	}
}

func collectBranch(n *parse.BranchNode, seen map[string]bool) {
	collectVarRefs(n.Pipe, seen)
	collectVarRefs(n.List, seen)
	collectVarRefs(n.ElseList, seen)
}

// fillUndeclaredVars returns data with every variable t references but
// data's Vars map lacks set to "", which renders as empty text and is false
// in an if. Without this, text/template prints a missing map key as
// "<no value>". Each such variable is warned about once per Renderer.
//
// data is left untouched unless it is a struct with a Vars
// map[string]any field; the caller's map is never modified.
func (r Renderer) fillUndeclaredVars(t *tmplpkg.Template, data any) any {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Struct {
		return data
	}
	field := v.FieldByName("Vars")
	if !field.IsValid() || field.Type() != reflect.TypeFor[map[string]any]() {
		return data
	}
	vars, _ := field.Interface().(map[string]any)

	var missing []string
	for _, name := range varRefs(t) {
		if _, ok := vars[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return data
	}

	filled := maps.Clone(vars)
	if filled == nil {
		filled = map[string]any{}
	}
	for _, name := range missing {
		filled[name] = ""
		r.warnUndeclared(t.Name(), name)
	}

	cp := reflect.New(v.Type()).Elem()
	cp.Set(v)
	cp.FieldByName("Vars").Set(reflect.ValueOf(filled))
	return cp.Interface()
}

func (r Renderer) warnUndeclared(templateName, name string) {
	if r.warned != nil {
		if r.warned[name] {
			return
		}
		r.warned[name] = true
	}
	var out io.Writer = os.Stderr
	if r.Warnings != nil {
		out = r.Warnings
	}
	fmt.Fprintf(out, "warning: %s uses .Vars.%s, which the template does not declare in %s; rendering it as empty\n", templateName, name, ManifestFileName)
}
