package main

import "testing"

func TestFixerFQCN(t *testing.T) {
	cases := []struct {
		path, prefix, want string
	}{
		{"Alias/ArrayPushFixer.php", `PhpCsFixer\Fixer\`, `PhpCsFixer\Fixer\Alias\ArrayPushFixer`},
		{"Commenting/LineLengthFixer.php", `Symplify\CodingStandard\Fixer\`, `Symplify\CodingStandard\Fixer\Commenting\LineLengthFixer`},
		{"AbstractPhpUnitFixer.php", `PhpCsFixer\Fixer\`, ""},
		{"Basic/AbstractFixer.php", `PhpCsFixer\Fixer\`, ""},
		{"IncrementOperatorFixer.php", `PhpCsFixer\Fixer\`, ""},
		{"Alias/ArrayPushSniff.php", `PhpCsFixer\Fixer\`, ""},
	}
	for _, c := range cases {
		if got := fixerFQCN(c.path, c.prefix); got != c.want {
			t.Errorf("fixerFQCN(%q, %q) = %q, want %q", c.path, c.prefix, got, c.want)
		}
	}
}

func TestSniffRe(t *testing.T) {
	src := `use PHP_CodeSniffer\Standards\Generic\Sniffs\WhiteSpace\LanguageConstructSpacingSniff;
use PHP_CodeSniffer\Sniffs\Sniff;
$x = SomeOtherSniff::class;`
	got := sniffRe.FindAllString(src, -1)
	if len(got) != 2 {
		t.Fatalf("expected 2 matches, got %d: %v", len(got), got)
	}
	if got[0] != `PHP_CodeSniffer\Standards\Generic\Sniffs\WhiteSpace\LanguageConstructSpacingSniff` {
		t.Errorf("unexpected first match: %q", got[0])
	}
}
