// SPDX-License-Identifier: GPL-3.0-or-later
package commands

import (
	"strings"
	"testing"
)

// No selection flags means every check runs, in the fixed syntax -> lint ->
// rubocop order (mirroring `pdk validate`). Any combination of flags narrows
// the run to exactly the selected checks, preserving that order.
func TestValidateTasks(t *testing.T) {
	cases := []struct {
		syntax, lint, rubocop bool
		want                  []string
	}{
		{false, false, false, []string{"validate", "lint", "rubocop"}},
		{true, false, false, []string{"validate"}},
		{false, true, false, []string{"lint"}},
		{false, false, true, []string{"rubocop"}},
		{true, true, false, []string{"validate", "lint"}},
		{true, false, true, []string{"validate", "rubocop"}},
		{false, true, true, []string{"lint", "rubocop"}},
		{true, true, true, []string{"validate", "lint", "rubocop"}},
	}

	for _, c := range cases {
		name := strings.Join(c.want, "+")
		t.Run(name, func(t *testing.T) {
			got := validateTasks(c.syntax, c.lint, c.rubocop)
			if len(got) != len(c.want) {
				t.Fatalf("validateTasks(%v, %v, %v) = %#v, want %#v",
					c.syntax, c.lint, c.rubocop, got, c.want)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Fatalf("validateTasks(%v, %v, %v) = %#v, want %#v",
						c.syntax, c.lint, c.rubocop, got, c.want)
				}
			}
		})
	}
}
