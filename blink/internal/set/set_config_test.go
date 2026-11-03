package set

import (
	"testing"

	"blink/internal/fixer"
	"blink/internal/lexer"
	"blink/internal/tokens"
)

// member returns the fixer of fixers whose short class name is short.
func member(t *testing.T, fixers []fixer.Fixer, short string) fixer.Fixer {
	t.Helper()
	for _, f := range fixers {
		if shortName(f.Name()) == short {
			return f
		}
	}
	t.Fatalf("%s is not a member", short)
	return nil
}

func fix(f fixer.Fixer, src string) string {
	s := tokens.New(lexer.Lex(src))
	f.Fix(s)
	return s.Render()
}

// TestSetsKeepECSRuleConfiguration checks that a set runs its members with the
// options ECS's prepared set gives them, not with PHP-CS-Fixer's defaults.
func TestSetsKeepECSRuleConfiguration(t *testing.T) {
	cases := []struct {
		set     string
		fixers  []fixer.Fixer
		rule    string
		in, out string
	}{
		{"common", Common(), "YodaStyleFixer", "<?php\nif (null === $x) {}", "<?php\nif ($x === null) {}"},
		{"default", Default(), "YodaStyleFixer", "<?php\nif (null === $x) {}", "<?php\nif ($x === null) {}"},
		{"per-cs", PERCS(), "YodaStyleFixer", "<?php\nif (null === $x) {}", "<?php\nif ($x === null) {}"},
		{"common", Common(), "GeneralPhpdocAnnotationRemoveFixer", "<?php\n/**\n * @author Foo\n * @see X\n */\nclass A {}", "<?php\n/**\n * @see X\n */\nclass A {}"},
		{"common", Common(), "ClassDefinitionFixer", "<?php\ninterface I extends J,\n    K\n{\n}", "<?php\ninterface I extends J, K\n{\n}"},
		{"per-cs", PERCS(), "ClassDefinitionFixer", "<?php\ninterface I extends J,\n    K\n{\n}\n$a = new class(\n    $b\n) {};", "<?php\ninterface I extends J,\n    K\n{\n}\n$a = new class (\n    $b\n) {};"},
		{"psr12", PSR12(), "ClassDefinitionFixer", "<?php\n$a = new class(\n    $b\n) {};", "<?php\n$a = new class($b) {};"},
		{"per-cs", PERCS(), "SingleSpaceAroundConstructFixer", "<?php\nif ($a) {\n}\nelse {\n}\necho  1;", "<?php\nif ($a) {\n} else {\n}\necho  1;"},
		{"psr12", PSR12(), "SingleSpaceAroundConstructFixer", "<?php\nif ($a) {\n}\nelse {\n}\necho  1;", "<?php\nif ($a) {\n}\nelse {\n}\necho 1;"},
		{"per-cs", PERCS(), "OrderedClassElementsFixer", "<?php\nclass A {\n    public function f() {}\n    private $a;\n    use T;\n}", "<?php\nclass A {\n    use T;\n    public function f() {}\n    private $a;\n}"},
		{"psr12", PSR12(), "ConcatSpaceFixer", "<?php\n$a = 'x'.'y';", "<?php\n$a = 'x' . 'y';"},
		{"spaces", Spaces(), "ConcatSpaceFixer", "<?php\n$a = 'x'.'y';", "<?php\n$a = 'x' . 'y';"},
		{"per-cs", PERCS(), "ConcatSpaceFixer", "<?php\n$a = 'x'.'y';", "<?php\n$a = 'x' . 'y';"},
		{"psr12", PSR12(), "OrderedImportsFixer", "<?php\nuse function Aaa\\f;\nuse Bbb\\B;\n", "<?php\nuse Bbb\\B;\nuse function Aaa\\f;\n"},
		{"per-cs", PERCS(), "UnaryOperatorSpacesFixer", "<?php\n$a = - $b;\n$e ++;", "<?php\n$a = - $b;\n$e++;"},
		{"psr12", PSR12(), "UnaryOperatorSpacesFixer", "<?php\n$a = - $b;\n$e ++;", "<?php\n$a = -$b;\n$e++;"},
	}
	for _, c := range cases {
		if got := fix(member(t, c.fixers, c.rule), c.in); got != c.out {
			t.Errorf("%s set, %s: got %q want %q", c.set, c.rule, got, c.out)
		}
	}
}
