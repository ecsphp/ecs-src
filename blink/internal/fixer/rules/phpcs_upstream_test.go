package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// phpcsMaxLoops mirrors PHP_CodeSniffer's Fixer, which re-runs the sniff until
// the file is stable or 50 loops pass.
const phpcsMaxLoops = 50

// TestPhpcsUpstreamFixtures runs every blink sniff over PHP_CodeSniffer's own
// test files (testdata/phpcs, copied from the 4.x Tests/ dirs): each .inc must
// fix to its .inc.fixed, or stay unchanged when upstream ships no .fixed file.
func TestPhpcsUpstreamFixtures(t *testing.T) {
	t.Parallel()
	for _, f := range All() {
		rest, ok := strings.CutPrefix(f.Name(), `PHP_CodeSniffer\Standards\`)
		if !ok {
			continue
		}
		// Generic\Sniffs\WhiteSpace\LanguageConstructSpacingSniff -> Generic/WhiteSpace/LanguageConstructSpacingUnitTest
		parts := strings.Split(strings.TrimSuffix(rest, "Sniff"), `\`)
		if len(parts) != 4 {
			t.Fatalf("unexpected sniff name %s", f.Name())
		}
		base := filepath.Join("testdata", "phpcs", parts[0], parts[2], parts[3]+"UnitTest")
		inputs, _ := filepath.Glob(base + ".inc")
		more, _ := filepath.Glob(base + ".*.inc")
		inputs = append(inputs, more...)
		if len(inputs) == 0 {
			t.Errorf("%s: no upstream fixtures at %s*.inc", f.Name(), base)
			continue
		}
		for _, in := range inputs {
			t.Run(filepath.Base(in), func(t *testing.T) {
				src := readFixture(t, in)
				want := src
				if _, err := os.Stat(in + ".fixed"); err == nil {
					want = readFixture(t, in+".fixed")
				}
				got := src
				for range phpcsMaxLoops {
					next, changed := apply(t, f, got)
					if !changed || next == got {
						break
					}
					got = next
				}
				if got != want {
					t.Errorf("%s: %s\n%s", f.Name(), in, firstDiff(got, want))
				}
			})
		}
	}
}

func readFixture(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// firstDiff shows the first differing line, enough to locate a fixture mismatch.
func firstDiff(got, want string) string {
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := range max(len(g), len(w)) {
		var gl, wl string
		if i < len(g) {
			gl = g[i]
		}
		if i < len(w) {
			wl = w[i]
		}
		if gl != wl {
			return fmt.Sprintf("line %d:\n  got:  %q\n  want: %q", i+1, gl, wl)
		}
	}
	return "outputs differ only in length"
}
