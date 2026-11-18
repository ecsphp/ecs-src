package rules

import "testing"

// malformed input once drove the scope-pop loop past the root scope and panicked
// with index -1; Fix must now return without panicking on any input.
func TestStatementIndentationNoPanicOnMalformed(t *testing.T) {
	t.Parallel()
	for _, src := range []string{"<?0(", "<?php }", "<?php {{{", "<?php case:"} {
		apply(t, StatementIndentation{}, src) // must not panic
	}
}

// A promoted constructor parameter with a leading attribute is still a param, not
// a class property, so the params after it must not gain an extra indent level.
func TestStatementIndentationAttributedPromotedParam(t *testing.T) {
	t.Parallel()
	src := "<?php\n\nclass E\n{\n    public function __construct(\n        public $guard,\n        #[\\SensitiveParameter]\n        public $credentials,\n        public $remember,\n    ) {\n    }\n}\n"
	if got, changed := apply(t, StatementIndentation{}, src); changed || got != src {
		t.Fatalf("attributed promoted param must not be re-indented; changed=%v got=%q", changed, got)
	}
}

// A PHP 8.4 "final" promoted constructor property is still a param: "final"
// before the visibility must not make it a class property, or the params after
// it and the closing ") {" gain an extra indent level.
func TestStatementIndentationFinalPromotedParam(t *testing.T) {
	t.Parallel()
	src := "<?php\n\nfinal class Flag\n{\n    public function __construct(\n        final public bool $on = false,\n        public int $level = 0,\n    ) {}\n}\n\nclass Box\n{\n    public function __construct(\n        public int $size = 0,\n        #[Marker]\n        final protected bool $open = false,\n    ) {\n    }\n}\n"
	if got, changed := apply(t, StatementIndentation{}, src); changed || got != src {
		t.Fatalf("final promoted param must not be re-indented; changed=%v got=%q", changed, got)
	}
}

// A comment that a blank line separates from the previous case body, or that sits
// right below another case label, describes the next case and takes its indent.
// A comment with no blank line above, and a "no break" marker, stay in the body.
func TestStatementIndentationCommentAboveCase(t *testing.T) {
	t.Parallel()
	src := "<?php\nswitch ($a) {\n    case 1:\n        return 'one';\n\n        // Two and three share a branch\n    case 2:\n    case 3:\n        // Three too\n    case 4:\n        return 'more';\n        // about this body\n    case 5:\n        foo();\n\n        // no break\n    default:\n        return null;\n}\n"
	want := "<?php\nswitch ($a) {\n    case 1:\n        return 'one';\n\n    // Two and three share a branch\n    case 2:\n    case 3:\n    // Three too\n    case 4:\n        return 'more';\n        // about this body\n    case 5:\n        foo();\n\n        // no break\n    default:\n        return null;\n}\n"
	for _, f := range []StatementIndentation{{}, {stickComment: true}} {
		if got, _ := apply(t, f, src); got != want {
			t.Fatalf("stickComment=%v: got=%q want=%q", f.stickComment, got, want)
		}
		if _, changed := apply(t, f, want); changed {
			t.Fatalf("stickComment=%v: fixed output must be stable", f.stickComment)
		}
	}
}

// Once a file has HTML in it, its PHP is aligned with the markup, which the
// scopes cannot see: that indent is left alone. The code before the first HTML
// is still reindented, and a shebang line is not HTML.
func TestStatementIndentationTemplate(t *testing.T) {
	t.Parallel()
	tpl := "<?php\nrequire 'x.php';\n?>\n<ul>\n    <?php foreach ($items as $item) { ?>\n        <li>\n            <?php\n            if ($item) {\n                echo $item;\n            }\n            ?>\n        </li>\n    <?php } ?>\n</ul>\n"
	if got, changed := apply(t, StatementIndentation{}, tpl); changed {
		t.Fatalf("template indent must be kept: got=%q", got)
	}
	got, changed := apply(t, StatementIndentation{}, "<?php\n    $a = 1;\n?>\n<p>\n        <?php echo $a; ?>\n</p>\n")
	if want := "<?php\n$a = 1;\n?>\n<p>\n        <?php echo $a; ?>\n</p>\n"; !changed || got != want {
		t.Fatalf("code before the HTML: changed=%v got=%q want=%q", changed, got, want)
	}
	got, changed = apply(t, StatementIndentation{}, "#!/usr/bin/env php\n<?php\nif ($a) {\nfoo();\n}\n")
	if want := "#!/usr/bin/env php\n<?php\nif ($a) {\n    foo();\n}\n"; !changed || got != want {
		t.Fatalf("shebang: changed=%v got=%q want=%q", changed, got, want)
	}
}

// Outputs of PHP-CS-Fixer's StatementIndentationFixer: whitespace-only lines
// before the last newline, alternative syntax blocks, and CRLF blank lines in
// block comments.
func TestStatementIndentationMatchesPhpCsFixer(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, src, want string }{
		{"blankLineBeforeClose", "<?php\nclass A {\n\tpublic function f() {\n\t\treturn 1;\n\t}\n\t\n}\n", "<?php\nclass A {\n    public function f() {\n        return 1;\n    }\n\n}\n"},
		{"twoBlankLines", "<?php\nif ($a) {\n    foo();\n    \n    \n}\n", "<?php\nif ($a) {\n    foo();\n    \n\n}\n"},
		{"altForeach", "<?php\nforeach ($a as $b):\n\n\t$c = 1;\n\tif ($c)\n\t\t$d = 2;\nendforeach;\n", "<?php\nforeach ($a as $b):\n\n    $c = 1;\n    if ($c)\n        $d = 2;\nendforeach;\n"},
		{"altIfElse", "<?php\nif ($a):\nfoo();\nelseif ($b):\nbar();\nelse:\nbaz();\nendif;\n", "<?php\nif ($a):\n    foo();\nelseif ($b):\n    bar();\nelse:\n    baz();\nendif;\n"},
		{"altNested", "<?php\nforeach ($a as $b):\nif ($b):\nfoo();\nendif;\nforeach ($b as $c):\nbar();\nendforeach;\nendforeach;\n", "<?php\nforeach ($a as $b):\n    if ($b):\n        foo();\n    endif;\n    foreach ($b as $c):\n        bar();\n    endforeach;\nendforeach;\n"},
		{"altSwitch", "<?php\nfunction f() {\nswitch ($a) {\ncase 1:\nswitch ($b):\ncase 2:\nfoo();\nendswitch;\nbreak;\n}\n}\n", "<?php\nfunction f() {\n    switch ($a) {\n        case 1:\n            switch ($b):\n                case 2:\n                    foo();\n            endswitch;\n            break;\n    }\n}\n"},
		{"altLoops", "<?php\nwhile ($a):\nfoo();\nendwhile;\nfor ($i = 0; $i < 3; $i++):\nfoo();\nendfor;\ndo {\nfoo();\n} while ($a);\n", "<?php\nwhile ($a):\n    foo();\nendwhile;\nfor ($i = 0; $i < 3; $i++):\n    foo();\nendfor;\ndo {\n    foo();\n} while ($a);\n"},
		{"notAlt", "<?php\nenum Suit: string {\ncase Hearts = 'H';\n}\n$f = function () use ($a): void {\nfoo();\n};\n", "<?php\nenum Suit: string {\n    case Hearts = 'H';\n}\n$f = function () use ($a): void {\n    foo();\n};\n"},
		{"crlfComment", "<?php\nif ($a) {\r\n\t\t/* x\r\n\t\t\r\n\t\ty */\r\n\t\tfoo();\r\n}\r\n", "<?php\nif ($a) {\r\n    /* x\r\n\t\t\r\n    y */\r\n    foo();\r\n}\r\n"},
	}
	for _, c := range cases {
		if got, _ := apply(t, StatementIndentation{}, c.src); got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
