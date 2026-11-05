package rules

import "testing"

// Cases verified against PHP_CodeSniffer 4.0.4: every expected output is what
// `phpcbf` produces for the input with only Squiz.WhiteSpace.SuperfluousWhitespace
// enabled.
func TestSuperfluousWhitespace(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, src, want string }{
		// end of line: whitespace, line comments, block comments, docblocks
		{"trailing spaces and tab", "<?php\n$a = 1;   \n$b = 2;\t\n", "<?php\n$a = 1;\n$b = 2;\n"},
		{"line comment trailing", "<?php\n$a = 1; // hi   \n$b = 2;\n", "<?php\n$a = 1; // hi\n$b = 2;\n"},
		{"hash comment trailing", "<?php\n# hash   \n$a=1;\n", "<?php\n# hash\n$a=1;\n"},
		{"line comment last token", "<?php\n// hi   ", "<?php\n// hi\n"},
		{"block comment internal line", "<?php\n/* a   \n   b   */\n$c = 3;\n", "<?php\n/* a\n   b   */\n$c = 3;\n"},
		{"docblock lines", "<?php\n/**   \n * x   \n */\nfunction f(){}\n", "<?php\n/**\n * x\n */\nfunction f(){}\n"},
		{"eof trailing no newline", "<?php\n$a = 1;   ", "<?php\n$a = 1;\n"},
		{"eof newline then trailing", "<?php\n$a = 1;   \n   ", "<?php\n$a = 1;\n\n"},
		{"eof clean no newline unchanged", "<?php\n$a = 1;", "<?php\n$a = 1;"},
		{"trailing blank lines unchanged", "<?php\n$a = 1;\n\n\n", "<?php\n$a = 1;\n\n\n"},
		{"trailing whitespace before close tag kept", "<?php\n$a = 1;   ?>\n", "<?php\n$a = 1;   ?>\n"},

		// start of file
		{"leading whitespace before php", "   \n<?php\n$a = 1;\n", "<?php\n$a = 1;\n"},
		{"leading real html kept", "x\n<?php\n$a = 1;\n", "x\n<?php\n$a = 1;\n"},
		{"echo tag start kept", "   \n<?= 1; ?>\n", "   \n<?= 1; ?>\n"},

		// end of file
		{"close tag spaces then newline", "<?php\necho 1;\n?>   \n", "<?php\necho 1;\n?>"},
		{"close tag newline then blanks", "<?php\necho 1;\n?>\n   \n   \n", "<?php\necho 1;\n?>\n"},
		{"close tag last token unchanged", "<?php\necho 1;\n?>\n", "<?php\necho 1;\n?>\n"},

		// multiple blank lines inside functions and closures
		{"one blank line kept", "<?php\nfunction f()\n{\n    $a = 1;\n\n    $b = 2;\n}\n", "<?php\nfunction f()\n{\n    $a = 1;\n\n    $b = 2;\n}\n"},
		{"collapse blank lines in function", "<?php\nfunction f(){\n$a=1;\n   \n   \n$b=2;\n}\n", "<?php\nfunction f(){\n$a=1;\n\n$b=2;\n}\n"},
		{"collapse after opening brace", "<?php\nfunction f()\n{\n\n\n\n    $a = 1;\n}\n", "<?php\nfunction f()\n{\n\n    $a = 1;\n}\n"},
		{"collapse before closing brace", "<?php\nfunction f()\n{\n    $a = 1;\n\n\n\n}\n", "<?php\nfunction f()\n{\n    $a = 1;\n\n}\n"},
		{"collapse in closure", "<?php\n$fn = function () {\n    $a = 1;\n\n\n\n    $b = 2;\n};\n", "<?php\n$fn = function () {\n    $a = 1;\n\n    $b = 2;\n};\n"},

		// scope: anonymous-class bodies and top level are left alone
		{"anon class body kept", "<?php\nfunction foo()\n{\n    $x = new class {\n        public $a = 1;\n\n\n\n        public $b = 2;\n    };\n}\n", "<?php\nfunction foo()\n{\n    $x = new class {\n        public $a = 1;\n\n\n\n        public $b = 2;\n    };\n}\n"},
		{"top level blank lines kept", "<?php\n$a=1;\n\n\n$b=2;\n", "<?php\n$a=1;\n\n\n$b=2;\n"},
		{"class body blank lines kept, method collapses", "<?php\nclass A\n{\n    public function m()\n    {\n        $a = 1;\n\n\n\n        $b = 2;\n    }\n\n\n\n    public function n()\n    {\n    }\n}\n", "<?php\nclass A\n{\n    public function m()\n    {\n        $a = 1;\n\n        $b = 2;\n    }\n\n\n\n    public function n()\n    {\n    }\n}\n"},

		// combined: top-level blank lines keep count but lose trailing whitespace
		{"top level strips blank-line whitespace, keeps count", "<?php\n$a=1;\n   \n   \n$b=2;\n", "<?php\n$a=1;\n\n\n$b=2;\n"},
	}

	for _, tc := range cases {
		got, _ := apply(t, SuperfluousWhitespace{}, tc.src)
		if got != tc.want {
			t.Errorf("%s:\n got %q\nwant %q", tc.name, got, tc.want)
		}
		idempotent(t, SuperfluousWhitespace{}, got)
	}
}

func TestSuperfluousWhitespaceSourceURL(t *testing.T) {
	t.Parallel()
	const want = "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/SuperfluousWhitespaceSniff.php"
	if got := (SuperfluousWhitespace{}).SourceURL(); got != want {
		t.Fatalf("SourceURL %q, want %q", got, want)
	}
}
