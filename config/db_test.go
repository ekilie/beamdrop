package config

import (
	"path/filepath"
	"testing"
)

func TestResolveDBPath(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty path",
			in:   "",
			want: "",
		},
		{
			name: "current directory trailing slash",
			in:   "./",
			want: filepath.Join(".", DBName),
		},
		{
			name: "current directory dot",
			in:   ".",
			want: filepath.Join(".", DBName),
		},
		{
			name: "explicit file path",
			in:   "./custom.db",
			want: "./custom.db",
		},
		{
			name: "existing directory path",
			in:   tempDir,
			want: filepath.Join(tempDir, DBName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveDBPath(tt.in)
			if got != tt.want {
				t.Fatalf("resolveDBPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
