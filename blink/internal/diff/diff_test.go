package diff

import (
	"strings"
	"testing"
)

func TestUnifiedEqual(t *testing.T) {
	if got := Unified("a\n", "a\n"); got != "" {
		t.Fatalf("equal inputs must yield empty diff, got %q", got)
	}
}

func TestUnifiedChange(t *testing.T) {
	got := Unified("<?php $x = 1 ;\n", "<?php $x = 1;\n")
	if !strings.Contains(got, "--- Original") || !strings.Contains(got, "+++ New") {
		t.Fatalf("missing header:\n%s", got)
	}
	if !strings.Contains(got, "@@ @@") {
		t.Fatalf("missing hunk header:\n%s", got)
	}
	if !strings.Contains(got, "-<?php $x = 1 ;") {
		t.Fatalf("missing removed line:\n%s", got)
	}
	if !strings.Contains(got, "+<?php $x = 1;") {
		t.Fatalf("missing added line:\n%s", got)
	}
}
