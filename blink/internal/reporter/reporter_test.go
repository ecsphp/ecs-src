package reporter

import (
	"bytes"
	"strings"
	"testing"

	"blink/internal/runner"
)

func TestReportClean(t *testing.T) {
	var buf bytes.Buffer
	code := Report(&buf, nil, false)
	if code != 0 {
		t.Fatalf("clean exit code = %d, want 0", code)
	}
	if !strings.Contains(buf.String(), "No errors found. Great job - your code is shiny in style!") {
		t.Fatalf("missing success message:\n%s", buf.String())
	}
}

func TestReportCheckFixable(t *testing.T) {
	var buf bytes.Buffer
	results := []runner.FileResult{{
		Path:         "src/Foo.php",
		AppliedRules: []string{`PhpCsFixer\Fixer\Semicolon\SpaceAfterSemicolonFixer`},
		Diff:         "--- Original\n+++ New\n@@ @@\n-$a=1;$b=2;\n+$a=1; $b=2;\n",
	}}
	code := Report(&buf, results, false)
	out := buf.String()
	if code != 1 {
		t.Fatalf("check exit code = %d, want 1", code)
	}
	for _, want := range []string{
		"1) src/Foo.php",
		"begin diff",
		"Applied checkers:",
		`PhpCsFixer\Fixer\Semicolon\SpaceAfterSemicolonFixer`,
		"1 error is fixable!",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestReportFixed(t *testing.T) {
	var buf bytes.Buffer
	results := []runner.FileResult{{Path: "a.php", AppliedRules: []string{"X"}}, {Path: "b.php", AppliedRules: []string{"Y"}}}
	code := Report(&buf, results, true)
	if code != 0 {
		t.Fatalf("fix exit code = %d, want 0", code)
	}
	if !strings.Contains(buf.String(), "2 errors successfully fixed and no other errors found!") {
		t.Fatalf("missing fixed message:\n%s", buf.String())
	}
}
