//go:build !windows

package okf

import "path/filepath"

func resolveExistingPath(name string) (string, error) {
	return filepath.EvalSymlinks(name)
}
