package reporter

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/runner"
)

func TestReportJSONClean(t *testing.T) {
	var buf bytes.Buffer
	if code := ReportJSON(&buf, nil, false); code != 0 {
		t.Fatalf("clean exit code = %d, want 0", code)
	}
	want := "{\n    \"totals\": {\n        \"errors\": 0,\n        \"diffs\": 0\n    },\n    \"files\": []\n}\n"
	if buf.String() != want {
		t.Fatalf("clean JSON:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestReportJSONMatchesECS(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cwd, err = filepath.EvalSymlinks(cwd)
	if err != nil {
		t.Fatal(err)
	}
	results := []runner.FileResult{
		{
			Path:         "src/A.php",
			AppliedRules: []string{`PhpCsFixer\Fixer\Semicolon\SpaceAfterSemicolonFixer`},
			Diff:         "--- Original\n+++ New\n@@ @@\n-$a=1;$b=2;\n+$a=1; $b=2;\n",
		},
		{
			Path: "src/B.php",
			Errors: []fixer.CodingStandardError{
				{Line: 9, Message: "second", Checker: `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\ForbiddenFunctionsSniff.Found`},
				{Line: 3, Message: "<first> & \"quoted\"", Checker: `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\ForbiddenFunctionsSniff.Found`},
			},
		},
	}

	var buf bytes.Buffer
	if code := ReportJSON(&buf, results, false); code != 1 {
		t.Fatalf("check exit code = %d, want 1", code)
	}

	a := jsonString(t, filepath.Join(cwd, "src", "A.php"))
	b := jsonString(t, filepath.Join(cwd, "src", "B.php"))
	want := `{
    "totals": {
        "errors": 2,
        "diffs": 1
    },
    "files": {
        ` + b + `: {
            "errors": [
                {
                    "line": 3,
                    "file_path": ` + b + `,
                    "message": "<first> & \"quoted\"",
                    "source_class": "PHP_CodeSniffer\\Standards\\Generic\\Sniffs\\PHP\\ForbiddenFunctionsSniff.Found"
                },
                {
                    "line": 9,
                    "file_path": ` + b + `,
                    "message": "second",
                    "source_class": "PHP_CodeSniffer\\Standards\\Generic\\Sniffs\\PHP\\ForbiddenFunctionsSniff.Found"
                }
            ]
        },
        ` + a + `: {
            "diffs": [
                {
                    "diff": "--- Original\n+++ New\n@@ @@\n-$a=1;$b=2;\n+$a=1; $b=2;\n",
                    "applied_checkers": [
                        "PhpCsFixer\\Fixer\\Semicolon\\SpaceAfterSemicolonFixer"
                    ]
                }
            ]
        }
    }
}
`
	if buf.String() != want {
		t.Fatalf("JSON:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestReportJSONFixExitCode(t *testing.T) {
	results := []runner.FileResult{{
		Path:         "src/A.php",
		AppliedRules: []string{`PhpCsFixer\Fixer\Semicolon\SpaceAfterSemicolonFixer`},
		Diff:         "--- Original\n+++ New\n",
	}}
	var buf bytes.Buffer
	if code := ReportJSON(&buf, results, true); code != 0 {
		t.Fatalf("fix with no errors left: exit code = %d, want 0", code)
	}
	results[0].Errors = []fixer.CodingStandardError{{Line: 1, Message: "m", Checker: "C"}}
	if code := ReportJSON(&buf, results, true); code != 1 {
		t.Fatalf("fix with errors left: exit code = %d, want 1", code)
	}
}

// jsonString is s as a quoted JSON string, so Windows paths compare escaped.
func jsonString(t *testing.T, s string) string {
	t.Helper()
	encoded, err := marshalNoEscape(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}
