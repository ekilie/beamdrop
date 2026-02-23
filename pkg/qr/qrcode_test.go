package qr_test

import (
	"github.com/ekilie/beamdrop/pkg/qr"
	"testing"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		data     string
		filename string
		wantErr  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := qr.Generate(tt.data, tt.filename)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Generate() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Generate() succeeded unexpectedly")
			}
		})
	}
}
