// SPDX-License-Identifier: GPL-3.0-or-later
package build

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ignoreFileNames are the ignore files consulted when building a module
// archive, in order of precedence. This mirrors PDK's behavior.
var ignoreFileNames = []string{".pdkignore", ".pmtignore", ".gitignore"}

// readIgnoreFile returns the contents of the first ignore file found in dir,
// along with the name of the file that was read. Candidates are tried in the
// order given by ignoreFileNames. A missing file falls through to the next
// candidate; any other read error is returned immediately. If no candidate
// exists, an error naming all of them is returned.
func readIgnoreFile(dir string) ([]byte, string, error) {
	for _, name := range ignoreFileNames {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			return data, name, nil
		}
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		return nil, "", fmt.Errorf("failed to read %s: %w", name, err)
	}
	return nil, "", fmt.Errorf("no ignore file found: looked for %s, %s, %s in %s",
		ignoreFileNames[0], ignoreFileNames[1], ignoreFileNames[2], dir)
}
