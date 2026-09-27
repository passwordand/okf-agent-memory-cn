package okf

import (
	"path/filepath"
	"strings"
)

// IsAbsPath securely checks if a path is absolute across platforms, preventing
// path traversal evasion where Windows paths (e.g., C:\) are evaluated on POSIX,
// or POSIX paths (e.g., /etc) are evaluated on Windows.
func IsAbsPath(path string) bool {
	if filepath.IsAbs(path) {
		return true
	}
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return true
	}
	if len(path) >= 2 && path[1] == ':' && ((path[0] >= 'a' && path[0] <= 'z') || (path[0] >= 'A' && path[0] <= 'Z')) {
		return true
	}
	return false
}
