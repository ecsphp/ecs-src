package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPhpcsOperatorContext pins the sniffs built on phpcsOperatorContext to
// ECS output (testdata/phpcs_operator/<Standard>.<Sniff>.php) on DNF types and
// the by-reference closures PHPCS misreads inside a ternary.
func TestPhpcsOperatorContext(t *testing.T) {
	t.Parallel()
	src := readFixture(t, filepath.Join("testdata", "phpcs_operator", "input.php"))
	for _, f := range All() {
		rest, ok := strings.CutPrefix(f.Name(), `PHP_CodeSniffer\Standards\`)
		if !ok {
			continue
		}
		parts := strings.Split(strings.TrimSuffix(rest, "Sniff"), `\`)
		golden := filepath.Join("testdata", "phpcs_operator", parts[0]+"."+parts[len(parts)-1]+".php")
		if _, err := os.Stat(golden); err != nil {
			continue
		}
		t.Run(filepath.Base(golden), func(t *testing.T) {
			t.Parallel()
			got := src
			for range phpcsMaxLoops {
				next, changed := apply(t, f, got)
				if !changed || next == got {
					break
				}
				got = next
			}
			if want := readFixture(t, golden); got != want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}
