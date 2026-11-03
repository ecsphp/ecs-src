package reporter

import (
	"bytes"
	"strings"
	"testing"

	"blink/internal/fixer"
	"blink/internal/runner"
)

const globalChecker = `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\GlobalKeywordSniff.NotAllowed`

func globalError(line int, name string) fixer.CodingStandardError {
	return fixer.CodingStandardError{
		Line:    line,
		Message: `Use of the "global" keyword is forbidden; use "$GLOBALS['` + name + `']" instead`,
		Checker: globalChecker,
	}
}

// TestReportCheckErrors matches ECS's output for errors that need fixing by
// hand: the framed error table sorted by file and line, then the count.
func TestReportCheckErrors(t *testing.T) {
	t.Setenv("COLUMNS", "")
	var buf bytes.Buffer
	results := []runner.FileResult{
		{Path: "src/b.php", Errors: []fixer.CodingStandardError{globalError(5, "q")}},
		{Path: "src/a.php", Errors: []fixer.CodingStandardError{globalError(5, "x"), globalError(6, "z")}},
	}
	code := Report(&buf, results, false)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	sep := " " + strings.Repeat("-", 112) + "\n"
	block := func(at, name string) string {
		return sep + " " + at + "\n" + sep +
			` Use of the "global" keyword is forbidden; use "$GLOBALS['` + name + `']" instead` + "\n\n" +
			` Reported by: "` + globalChecker + `"` + "\n" + sep + "\n"
	}
	want := "\n\n\n" + block("src/a.php:5", "x") + block("src/a.php:6", "z") + block("src/b.php:5", "q") +
		"\n [ERROR] Found 3 errors that need to be fixed manually.\n"
	if got := buf.String(); got != want {
		t.Fatalf("output:\n%q\nwant:\n%q", got, want)
	}
}

// TestReportCheckErrorsAndDiffs adds ECS's "Good news" fixable line.
func TestReportCheckErrorsAndDiffs(t *testing.T) {
	var buf bytes.Buffer
	results := []runner.FileResult{
		{Path: "a.php", AppliedRules: []string{"X"}, Diff: "--- Original\n+++ New\n@@ @@\n-$a=1;\n+$a = 1;\n", Errors: []fixer.CodingStandardError{globalError(1, "w")}},
	}
	if code := Report(&buf, results, false); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	out := buf.String()
	for _, want := range []string{
		"1) a.php",
		" a.php:1\n",
		"\n [ERROR] Found 1 error that needs to be fixed manually.\n",
		"\n [WARNING] Good news is that 1 error is fixable! Just add \"--fix\" to console command and rerun to apply.\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

// TestReportFixWithErrors fails a fix run that leaves errors, without the
// success line.
func TestReportFixWithErrors(t *testing.T) {
	var buf bytes.Buffer
	results := []runner.FileResult{{Path: "a.php", Errors: []fixer.CodingStandardError{globalError(2, "w")}}}
	if code := Report(&buf, results, true); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	out := buf.String()
	if strings.Contains(out, "successfully fixed") || !strings.Contains(out, " [ERROR] Found 1 error that needs to be fixed manually.") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

// TestPhpWordwrap mirrors PHP's wordwrap($text, $width, "\n", false).
func TestPhpWordwrap(t *testing.T) {
	for _, c := range []struct {
		in    string
		width int
		want  string
	}{
		{"The quick brown fox", 10, "The quick\nbrown fox"},
		{"A very long woooooooooooord.", 8, "A very\nlong\nwoooooooooooord."},
		{"short\n\n Reported by: \"X\"", 112, "short\n\n Reported by: \"X\""},
		{`Use of the "global" keyword is forbidden; use "$GLOBALS['x']" instead`, 52, "Use of the \"global\" keyword is forbidden; use\n\"$GLOBALS['x']\" instead"},
	} {
		if got := phpWordwrap(c.in, c.width); got != c.want {
			t.Errorf("phpWordwrap(%q, %d) = %q, want %q", c.in, c.width, got, c.want)
		}
	}
}
