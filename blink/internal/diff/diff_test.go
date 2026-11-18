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
	if !strings.Contains(got, "@@ -1 +1 @@") {
		t.Fatalf("missing hunk header:\n%s", got)
	}
	if !strings.Contains(got, "-<?php $x = 1 ;") {
		t.Fatalf("missing removed line:\n%s", got)
	}
	if !strings.Contains(got, "+<?php $x = 1;") {
		t.Fatalf("missing added line:\n%s", got)
	}
}

func TestUnifiedHunkHeadersMatchPHPCSFixer(t *testing.T) {
	before := "<?php\n\n$a = 1;\n$b = 2 ;\n$c = 3;\n$d = 4;\n$e = 5;\n$f = 6;\n$g = 7;\n$h = 8;\n$i = 9;\n$j = 10 ;\n"
	after := "<?php\n\n$a = 1;\n$b = 2;\n$c = 3;\n$d = 4;\n$e = 5;\n$f = 6;\n$g = 7;\n$h = 8;\n$i = 9;\n// ten\n$j = 10;\n"
	want := "--- Original\n+++ New\n" +
		"@@ -1,7 +1,7 @@\n <?php\n \n $a = 1;\n-$b = 2 ;\n+$b = 2;\n $c = 3;\n $d = 4;\n $e = 5;\n" +
		"@@ -9,4 +9,5 @@\n $g = 7;\n $h = 8;\n $i = 9;\n-$j = 10 ;\n+// ten\n+$j = 10;\n"
	if got := Unified(before, after); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
