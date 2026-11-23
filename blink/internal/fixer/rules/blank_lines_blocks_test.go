package rules

import "testing"

func noExtraBlankLinesTokens(tokens ...string) map[string]any {
	list := make([]any, len(tokens))
	for i, t := range tokens {
		list[i] = t
	}
	return map[string]any{"tokens": list}
}

func noExtraBlankLinesWith(tokens ...string) fixerRule {
	return NoExtraBlankLines{}.WithConfig(noExtraBlankLinesTokens(tokens...)).(fixerRule)
}

// noExtraBlankLinesAllTokens is a configuration with every block token on; the
// expected outputs below are PHP-CS-Fixer's with it.
var noExtraBlankLinesAllTokens = []string{"attribute", "case", "continue", "curly_brace_block", "default", "extra", "parenthesis_brace_block", "square_brace_block", "switch", "throw", "use"}

// Cases of PHP-CS-Fixer's NoExtraBlankLinesFixerTest for the block and attribute tokens.
func TestNoExtraBlankLinesPhpCsFixerBlockCases(t *testing.T) {
	t.Parallel()
	oneOrInLine := noExtraBlankLinesTokens("break", "continue", "return", "throw", "curly_brace_block", "square_brace_block", "parenthesis_brace_block")
	curly := noExtraBlankLinesTokens("curly_brace_block")
	parenthesis := noExtraBlankLinesTokens("parenthesis_brace_block")
	attribute := noExtraBlankLinesTokens("attribute")
	runUpstreamCases(t, NoExtraBlankLines{}, []upstreamCase{
		{"closure on one line", oneOrInLine, "<?php\n\n$a = function() use ($b) { while(3<1)break; $c = $b[1]; while($b<1)continue; if (true) throw $e; return 1; };\n\n", ""},
		{"throw at the end of the file", oneOrInLine, "<?php throw new \\Exception('do not import.');\n", "<?php throw new \\Exception('do not import.');\n\n"},
		{"index access", oneOrInLine, "<?php\n\n$a = $b[0];\n\n", ""},
		{"dynamic property", oneOrInLine, "<?php\n\n$a->{'Test'};\nfunction test(){}\n", ""},
		{"anonymous class on one line", oneOrInLine, "<?php\n\n$a = new class { public function a () { while(4<1)break; while(3<1)continue; if (true) throw $e; return 1; }};\n\n", ""},
		{"blank line before an empty body", curly, "<?php function test()\n\n{}\n\necho 789;", ""},
		{"switch body", curly, "<?php switch($a){\ncase 1:echo 789;}", "<?php switch($a){\n   \ncase 1:echo 789;}"},
		{"call and parameter list", parenthesis, "<?php\nis_int(\n1);\nfunction test(\n$a,\n$b,\n$c\n)\n{\n\n\n}", "<?php\nis_int(\n\n1);\nfunction test(\n\n$a,\n$b,\n$c\n\n\n)\n{\n\n\n}"},
		{"array()", parenthesis, "<?php array(\n1,\n2,\n3,\n);", "<?php array(\n  \n1,\n2,\n3,\n\n\n);"},
		{"closing parenthesis on its own line", parenthesis, "<?php\n    function a()\n    {\n        $b->d(e(\n        ));\n\n        foreach ($a as $x) {\n        }\n    }", ""},
		{"short array", noExtraBlankLinesTokens("square_brace_block"), "<?php $c = $b[0];\n\n\n$a = [\n   1,\n2];\necho 1;\n$b = [];\n\n\n//a\n", "<?php $c = $b[0];\n\n\n$a = [\n\n   1,\n2];\necho 1;\n$b = [];\n\n\n//a\n"},
		{"blank lines after attributes", attribute, "<?php\n#[Attr]\n#[AttrFoo1]\n#[AttrFoo2]\nfunction foo(){}\n            ", "<?php\n#[Attr]\n\n\n\n#[AttrFoo1]\n\n\n#[AttrFoo2]\n\nfunction foo(){}\n            "},
		{"parameter attribute", attribute, "<?php class Foo\n            {\n                protected function f1(string $x): string { return \"r1\"; }\n\n                protected function f2(string $x): string { return \"r1\"; }\n\n                protected function f3(#[Attr] string $x): string { return \"r1\"; }\n\n                protected function f4(string $x): string { return \"r1\"; }\n            }", ""},
		{"parameter attribute of an abstract method", attribute, "<?php abstract class Foo\n            {\n                abstract protected function f1(string $x): string;\n\n                abstract protected function f2(string $x): string;\n\n                abstract protected function f3(#[Attr] string $x): string;\n\n                abstract protected function f4(string $x): string;\n            }", ""},
	})
}

func TestNoExtraBlankLinesParenthesisBraceBlock(t *testing.T) {
	t.Parallel()
	f := noExtraBlankLinesWith(noExtraBlankLinesAllTokens...)

	// blank lines after "(" and before ")" of multi-line parameter and argument lists
	assertFix(t, f,
		"<?php\n\nfinal readonly class Query\n{\n    public function __construct(\n\n        public ?string $date = null,\n\n        public string $dir = 'x',\n\n    ) {}\n\n    public static function make(): self\n    {\n        return new self(\n            date: 'today',\n\n        );\n    }\n}\n",
		"<?php\n\nfinal readonly class Query\n{\n    public function __construct(\n        public ?string $date = null,\n\n        public string $dir = 'x',\n    ) {}\n\n    public static function make(): self\n    {\n        return new self(\n            date: 'today',\n        );\n    }\n}\n", true)

	// comments after "(" or before ")" stay, the blank lines around them go
	assertFix(t, f,
		"<?php\nfoo( // c\n\n    $a,\n);\nbar(\n\n    // c\n    $b\n);\nbaz(\n    $c,\n    // c\n\n);\nqux(\n    $d // c\n\n);\n",
		"<?php\nfoo( // c\n    $a,\n);\nbar(\n    // c\n    $b\n);\nbaz(\n    $c,\n    // c\n);\nqux(\n    $d // c\n);\n", true)

	// closure parameters and use list, match subject, conditions, list(), array() and an empty pair
	assertFix(t, f,
		"<?php\nfoo(function (\n\n    $a\n\n) use (\n\n    $b\n\n) {\n\n    return 1;\n\n});\n$r = match (\n\n    $x\n\n) {\n    1 => 2,\n};\nif (\n\n    $a\n    && $b\n\n) {\n}\nwhile ($a\n\n    && $b) {\n}\nfor (\n\n    $i = 0;;\n\n) {\n}\nlist(\n\n    $c,\n\n) = $d;\n$e = array(\n\n    1\n\n);\nf(\n\n);\n",
		"<?php\nfoo(function (\n    $a\n) use (\n    $b\n) {\n    return 1;\n});\n$r = match (\n    $x\n) {\n    1 => 2,\n};\nif (\n    $a\n    && $b\n) {\n}\nwhile ($a\n    && $b) {\n}\nfor (\n    $i = 0;;\n) {\n}\nlist(\n    $c,\n) = $d;\n$e = array(\n    1\n);\nf(\n);\n", true)

	// the last line break of the block is cleaned even when it is not before ")"
	assertFix(t, f,
		"<?php\nfoo($a,\n\n    $b);\n",
		"<?php\nfoo($a,\n    $b);\n", true)

	// the parentheses of "(new Foo)" are not a parenthesis block
	assertFix(t, f,
		"<?php\n$a = (\n\n    new Foo()\n\n)->bar();\n$b = (new Foo(\n\n    1\n\n))->bar();\n",
		"<?php\n$a = (\n\n    new Foo()\n\n)->bar();\n$b = (new Foo(\n    1\n))->bar();\n", true)

	// the cleaned line break also loses the spaces before it
	assertFix(t, f,
		"<?php\nfoo(\n    $a,  \n);\n",
		"<?php\nfoo(\n    $a,\n);\n", true)

	// the parentheses of a DNF type are not a parenthesis block; a parameter
	// list around one is
	assertFix(t, f,
		"<?php\nclass C\n{\n    public (\n\n        A&B\n\n    )|null $p;\n\n    public function __construct(\n\n        public (A&B)|null $q = null,\n\n    ) {}\n}\nfunction f((\n\n    A&B\n\n)|null $x): (\n\n    C&D\n\n)|null {}\n",
		"<?php\nclass C\n{\n    public (\n\n        A&B\n\n    )|null $p;\n\n    public function __construct(\n        public (A&B)|null $q = null,\n    ) {}\n}\nfunction f((\n    A&B\n)|null $x): (\n\n    C&D\n\n)|null {}\n", true)
}

// The line after a token ends at the first token holding a line break, whatever
// its kind, or at a plain "}".
func TestNoExtraBlankLinesLineEnd(t *testing.T) {
	t.Parallel()
	f := noExtraBlankLinesWith(noExtraBlankLinesAllTokens...)

	// a multi-line string, comment or heredoc on the line of "(" ends that line
	assertFix(t, f,
		"<?php\nfoo('a\nb',\n\n    $c,\n);\nbar(/* a\nb */\n\n    $d,\n);\nbaz(<<<'EOT'\na\nEOT,\n\n    $e,\n);\n",
		"<?php\nfoo('a\nb',\n\n    $c,\n);\nbar(/* a\nb */\n\n    $d,\n);\nbaz(<<<'EOT'\na\nEOT,\n\n    $e,\n);\n", false)

	// so does a multi-line string after throw
	assertFix(t, noExtraBlankLinesWith("throw"),
		"<?php\nthrow new E('a\nb');\n\n$x = 1;\n",
		"<?php\nthrow new E('a\nb');\n\n$x = 1;\n", false)

	// a closure's "}" ends the line of "(", a dynamic access brace does not
	assertFix(t, f,
		"<?php\nfoo(function () { return 1; },\n\n    $b,\n\n);\nbar($c->{'d'},\n\n    $e,\n);\nbaz(${'f'},\n\n    $g,\n);\n",
		"<?php\nfoo(function () { return 1; },\n\n    $b,\n);\nbar($c->{'d'},\n    $e,\n);\nbaz(${'f'},\n    $g,\n);\n", true)
}

func TestNoExtraBlankLinesSquareBraceBlock(t *testing.T) {
	t.Parallel()
	f := noExtraBlankLinesWith(noExtraBlankLinesAllTokens...)

	// only short array brackets are blocks: not index access nor destructuring
	// brackets, except a nested one after "=>", which SquareBraceTransformer
	// takes for an array; a foreach destructuring is cleaned by its "("
	assertFix(t, f,
		"<?php\n$a = [\n\n    1,\n\n];\n[\n\n    $b,\n\n] = $c;\n$d = $e[\n\n    0\n\n];\n['j' => $f, 'k' => [\n\n    $g\n\n]] = $h;\n[$i, [\n\n    $j\n\n]] = $k;\nforeach ($l as [\n\n    $m\n\n]) {\n}\n",
		"<?php\n$a = [\n    1,\n];\n[\n\n    $b,\n\n] = $c;\n$d = $e[\n\n    0\n\n];\n['j' => $f, 'k' => [\n    $g\n]] = $h;\n[$i, [\n\n    $j\n\n]] = $k;\nforeach ($l as [\n    $m\n]) {\n}\n", true)
}

func TestNoExtraBlankLinesAttribute(t *testing.T) {
	t.Parallel()
	f := noExtraBlankLinesWith(noExtraBlankLinesAllTokens...)

	// blank lines after the line that closes an attribute go, unless an array
	// or a closure opens earlier on that line; the inside of "#[" is no block
	assertFix(t, f,
		"<?php\nclass A\n{\n    #[Foo]\n\n    public $a;\n\n    #[Bar] public $b;\n\n    public $c;\n\n    #[Baz([1])]\n\n    public $d;\n\n    #[\n\n        Qux\n\n    ]\n    public $e;\n}\n$f = function (#[A] $x) {};\n\n$g = #[A] fn ($x) => 1;\n\n$h = 1;\n",
		"<?php\nclass A\n{\n    #[Foo]\n    public $a;\n\n    #[Bar] public $b;\n    public $c;\n\n    #[Baz([1])]\n\n    public $d;\n\n    #[\n\n        Qux\n\n    ]\n    public $e;\n}\n$f = function (#[A] $x) {};\n\n$g = #[A] fn ($x) => 1;\n$h = 1;\n", true)
}

func TestNoExtraBlankLinesCurlyBraceKinds(t *testing.T) {
	t.Parallel()
	f := noExtraBlankLinesWith(noExtraBlankLinesAllTokens...)

	// property hook and dynamic class constant braces are no blocks; a brace
	// spaced from "->" and a dynamic static method name brace are
	assertFix(t, f,
		"<?php\nclass A\n{\n    public string $a {\n\n        get => 'a';\n\n    }\n}\n$b = $c-> {\n\n    'd'\n\n};\n$e = A::{\n\n    'F'\n\n};\n$g = A::{\n\n    'h'\n\n}();\n",
		"<?php\nclass A\n{\n    public string $a {\n\n        get => 'a';\n\n    }\n}\n$b = $c-> {\n    'd'\n};\n$e = A::{\n\n    'F'\n\n};\n$g = A::{\n    'h'\n}();\n", true)
}

// Each token only cleans its own blocks.
func TestNoExtraBlankLinesTokenGating(t *testing.T) {
	t.Parallel()
	src := "<?php\n#[A]\n\nfunction f(\n\n    $a,\n\n) {\n\n    return [\n\n        $a,\n\n    ];\n\n}\n"

	assertFix(t, NoExtraBlankLines{}, src, src, false)
	assertFix(t, noExtraBlankLinesWith("extra"), src, src, false)
	assertFix(t, noExtraBlankLinesWith("curly_brace_block", "use"), src,
		"<?php\n#[A]\n\nfunction f(\n\n    $a,\n\n) {\n    return [\n\n        $a,\n\n    ];\n}\n", true)
	assertFix(t, noExtraBlankLinesWith("parenthesis_brace_block"), src,
		"<?php\n#[A]\n\nfunction f(\n    $a,\n) {\n\n    return [\n\n        $a,\n\n    ];\n\n}\n", true)
	assertFix(t, noExtraBlankLinesWith("square_brace_block"), src,
		"<?php\n#[A]\n\nfunction f(\n\n    $a,\n\n) {\n\n    return [\n        $a,\n    ];\n\n}\n", true)
	assertFix(t, noExtraBlankLinesWith("attribute"), src,
		"<?php\n#[A]\nfunction f(\n\n    $a,\n\n) {\n\n    return [\n\n        $a,\n\n    ];\n\n}\n", true)
}
