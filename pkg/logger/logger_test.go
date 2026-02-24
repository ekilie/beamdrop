package logger

import "testing"

func TestSanitizeSourceFilePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "module path in source",
			in:   "/tmp/build/github.com/ekilie/beamdrop/pkg/db/db.go",
			want: "pkg/db/db.go",
		},
		{
			name: "beamdrop directory marker",
			in:   "/home/runner/work/beamdrop/beam/server/server.go",
			want: "beam/server/server.go",
		},
		{
			name: "fallback to last three segments",
			in:   "/opt/random/path/to/file.go",
			want: "path/to/file.go",
		},
		{
			name: "already short",
			in:   "pkg/logger/logger.go",
			want: "pkg/logger/logger.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeSourceFilePath(tt.in)
			if got != tt.want {
				t.Fatalf("sanitizeSourceFilePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
