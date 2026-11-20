package rules

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// phpcsMaxLoops mirrors PHP_CodeSniffer's Fixer, which re-runs the sniff until
// the file is stable or 50 loops pass.
const phpcsMaxLoops = 50

// phpcsOnePass is a sniff whose Fix already runs ECS's fixer loop through to a
// fixer conflict, where ECS stops; fixing its output again would go further.
type phpcsOnePass interface{ phpcsOnePass() }

// TestPhpcsUpstreamFixtures runs every blink sniff over PHP_CodeSniffer's own
// test files (testdata/phpcs, copied from the 4.x Tests/ dirs): each .inc must
// fix to its .inc.fixed, or stay unchanged when upstream ships no .fixed file.
func TestPhpcsUpstreamFixtures(t *testing.T) {
	t.Parallel()
	for _, f := range All() {
		inputs, ok := phpcsUpstreamInputs(t, f)
		if !ok {
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
					if _, ok := f.(phpcsOnePass); ok {
						break
					}
				}
				if got != want {
					t.Errorf("%s: %s\n%s", f.Name(), in, firstDiff(got, want))
				}
			})
		}
	}
}

// TestPhpcsUpstreamErrors holds every blink sniff that reports errors to the
// errors ECS reports on PHP_CodeSniffer's own test files: each .inc must report
// exactly the rows of its .inc.errors, or none when there is no such file.
func TestPhpcsUpstreamErrors(t *testing.T) {
	t.Parallel()
	for _, f := range All() {
		reporter, ok := f.(fixer.ErrorReporter)
		if _, fs := f.(phpcsReadsFileSystem); !ok || fs {
			continue
		}
		inputs, ok := phpcsUpstreamInputs(t, f)
		if !ok {
			continue
		}
		for _, in := range inputs {
			t.Run(filepath.Base(in), func(t *testing.T) {
				if _, ok := f.(PHPSyntax); ok {
					if _, err := exec.LookPath("php"); err != nil {
						t.Skip("Generic Syntax runs php -l, no php on PATH")
					}
				}
				var want []string
				if _, err := os.Stat(in + ".errors"); err == nil {
					want = strings.Split(strings.TrimSuffix(readFixture(t, in+".errors"), "\n"), "\n")
				}
				s := tokens.New(lexer.Lex(readFixture(t, in)))
				// bin/phpcs-fixtures.sh hands each input to ECS as <name>.inc.php
				s.SetPath(filepath.Base(in) + ".php")
				var got []string
				if c, ok := f.(fixer.Candidate); !ok || c.IsCandidate(s) {
					for _, e := range reporter.Errors(s) {
						got = append(got, phpcsErrorRow(e))
					}
				}
				slices.Sort(got)
				slices.Sort(want)
				if !slices.Equal(got, want) {
					t.Errorf("%s: %s\n got: %q\nwant: %q", f.Name(), in, got, want)
				}
			})
		}
	}
}

// phpcsReadsFileSystem is a sniff that reports from the file on disk rather
// than its contents, which TestPhpcsUpstreamErrors cannot reproduce.
type phpcsReadsFileSystem interface{ phpcsReadsFileSystem() }

// phpcsErrorRow formats an error as bin/phpcs-fixtures.sh writes it.
func phpcsErrorRow(e fixer.CodingStandardError) string {
	msg := strings.NewReplacer(`\`, `\\`, "\n", `\n`, "\t", `\t`).Replace(e.Message)
	return strconv.Itoa(e.Line) + "\t" + e.Checker + "\t" + msg
}

// phpcsUpstreamInputs lists the upstream test inputs of a blink sniff; ok is
// false for a rule that is no PHP_CodeSniffer sniff.
func phpcsUpstreamInputs(t *testing.T, f fixer.Fixer) ([]string, bool) {
	t.Helper()
	rest, ok := strings.CutPrefix(f.Name(), `PHP_CodeSniffer\Standards\`)
	if !ok {
		return nil, false
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
		return nil, false
	}
	return inputs, true
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
