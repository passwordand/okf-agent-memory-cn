package okf

import (
	"testing"
)

func TestIsAbsPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/etc/passwd", true},
		{"\\Windows\\System32", true},
		{"C:\\Windows", true},
		{"d:/temp", true},
		{"Z:\\", true},
		{"relative/path", false},
		{"./local/file", false},
		{"..\\up", false},
		{"file.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsAbsPath(tt.path); got != tt.want {
				t.Errorf("IsAbsPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
