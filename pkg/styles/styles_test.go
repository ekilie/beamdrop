package styles

import (
	"testing"
)

func TestStylesExist(t *testing.T) {
	if InfoStyle == nil {
		t.Error("InfoStyle should not be nil")
	}
	if WarningStyle == nil {
		t.Error("WarningStyle should not be nil")
	}
	if ErrorStyle == nil {
		t.Error("ErrorStyle should not be nil")
	}
	if DebugStyle == nil {
		t.Error("DebugStyle should not be nil")
	}
	if TitleStyle == nil {
		t.Error("TitleStyle should not be nil")
	}
}
