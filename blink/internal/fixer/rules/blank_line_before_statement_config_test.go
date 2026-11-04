package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

func TestBlankLineBeforeStatementConfig(t *testing.T) {
	t.Parallel()
	const loop = "<?php\nforeach ($items as $item) {\n    $a = 1;\n    continue;\n}\nfunction f()\n{\n    $a = 1;\n    return $a;\n}\nfunction g()\n{\n    $a = 1;\n    throw new E();\n}\n"

	// default statements (also what an empty config falls back to) cover continue,
	// return and throw
	const all = "<?php\nforeach ($items as $item) {\n    $a = 1;\n\n    continue;\n}\nfunction f()\n{\n    $a = 1;\n\n    return $a;\n}\nfunction g()\n{\n    $a = 1;\n\n    throw new E();\n}\n"
	for name, f := range map[string]fixerRule{
		"zero":  BlankLineBeforeStatement{},
		"empty": BlankLineBeforeStatement{}.WithConfig(map[string]any{}).(fixerRule),
	} {
		if got, changed := apply(t, f, loop); !changed || got != all {
			t.Errorf("%s: changed=%v got=%q", name, changed, got)
		}
	}

	// statements: [return] leaves continue and throw alone
	f := BlankLineBeforeStatement{}.WithConfig(map[string]any{"statements": []any{"return"}}).(fixerRule)
	want := "<?php\nforeach ($items as $item) {\n    $a = 1;\n    continue;\n}\nfunction f()\n{\n    $a = 1;\n\n    return $a;\n}\nfunction g()\n{\n    $a = 1;\n    throw new E();\n}\n"
	if got, changed := apply(t, f, loop); !changed || got != want {
		t.Errorf("return only: changed=%v got=%q", changed, got)
	}

	// statements: [] disables the rule
	f = BlankLineBeforeStatement{}.WithConfig(map[string]any{"statements": []any{}}).(fixerRule)
	if got, changed := apply(t, f, loop); changed || got != loop {
		t.Errorf("none: changed=%v got=%q", changed, got)
	}
}

func TestBlankLineBeforeStatementNonDefaultStatements(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, statement, in, want string
	}{
		{
			"while, not the one of a do-while", "while",
			"<?php\n$a = 1;\nwhile ($a) {\n    $a--;\n}\ndo {\n    $a++;\n} while ($a < 3);\n",
			"<?php\n$a = 1;\n\nwhile ($a) {\n    $a--;\n}\ndo {\n    $a++;\n} while ($a < 3);\n",
		},
		{
			"switch case, not an enum case", "case",
			"<?php\nenum Suit\n{\n    case Hearts;\n    case Spades;\n}\nswitch ($a) {\n    case 1:\n        $b = 1;\n    case 2:\n        $b = 2;\n}\n",
			"<?php\nenum Suit\n{\n    case Hearts;\n    case Spades;\n}\nswitch ($a) {\n    case 1:\n        $b = 1;\n\n    case 2:\n        $b = 2;\n}\n",
		},
		{
			"yield is not yield from", "yield",
			"<?php\nfunction f()\n{\n    yield 1;\n    yield from g();\n    yield 2;\n}\n",
			"<?php\nfunction f()\n{\n    yield 1;\n    yield from g();\n\n    yield 2;\n}\n",
		},
		{
			"yield from", "yield_from",
			"<?php\nfunction f()\n{\n    yield 1;\n    yield from g();\n}\n",
			"<?php\nfunction f()\n{\n    yield 1;\n\n    yield from g();\n}\n",
		},
		{
			// PHP >= 8.3 lexes "yield", any whitespace or comments, then "from" as one
			// T_YIELD_FROM, which the yield entry does not cover
			"yield is not yield from with a comment between", "yield",
			"<?php\nfunction f()\n{\n    yield 1;\n    yield /* c */ from g();\n    yield 2;\n    yield\n    // c\n    from g();\n    yield # c\n    from g();\n}\n",
			"<?php\nfunction f()\n{\n    yield 1;\n    yield /* c */ from g();\n\n    yield 2;\n    yield\n    // c\n    from g();\n    yield # c\n    from g();\n}\n",
		},
		{
			"yield from with a comment between", "yield_from",
			"<?php\nfunction f()\n{\n    yield 1;\n    yield /* c */ from g();\n    yield 2;\n    yield\n    // c\n    from g();\n    yield # c\n    from g();\n}\n",
			"<?php\nfunction f()\n{\n    yield 1;\n\n    yield /* c */ from g();\n    yield 2;\n\n    yield\n    // c\n    from g();\n\n    yield # c\n    from g();\n}\n",
		},
		{
			"exit and die", "exit",
			"<?php\n$a = 1;\nexit(0);\n$b = 1;\ndie();\n",
			"<?php\n$a = 1;\n\nexit(0);\n$b = 1;\n\ndie();\n",
		},
		{
			"phpdoc", "phpdoc",
			"<?php\n$a = 1;\n/** @var int $b */\n$b = f();\n",
			"<?php\n$a = 1;\n\n/** @var int $b */\n$b = f();\n",
		},
		{
			"if after a statement, not after else", "if",
			"<?php\n$a = 1;\nif ($a) {\n} else if ($b) {\n}\n",
			"<?php\n$a = 1;\n\nif ($a) {\n} else if ($b) {\n}\n",
		},
	}
	for _, c := range cases {
		f := BlankLineBeforeStatement{}.WithConfig(map[string]any{"statements": []any{c.statement}}).(fixerRule)
		got, _ := apply(t, f, c.in)
		if got != c.want {
			t.Errorf("%s:\n got=%q\nwant=%q", c.name, got, c.want)
		}
		if again, changed := apply(t, f, got); changed || again != got {
			t.Errorf("%s: not idempotent: %q", c.name, again)
		}
	}
}

// A lexer may emit "yield from" (with whatever sits between the words) as one
// token, like PHP's T_YIELD_FROM; it is still the yield_from statement.
func TestBlankLineBeforeStatementYieldFromSingleToken(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"yield from", "yield /* c */ from", "yield/**/from", "YIELD\nFROM"} {
		toks := []token.Token{
			{Kind: token.OpenTag, Value: "<?php\n"},
			{Kind: token.Variable, Value: "$a"},
			{Kind: token.Punct, Value: ";"},
			{Kind: token.Whitespace, Value: "\n"},
			{Kind: token.Keyword, Value: v},
			{Kind: token.Whitespace, Value: " "},
			{Kind: token.Ident, Value: "g"},
			{Kind: token.Punct, Value: "("},
			{Kind: token.Punct, Value: ")"},
			{Kind: token.Punct, Value: ";"},
		}
		for statement, want := range map[string]bool{"yield": false, "yield_from": true} {
			f := BlankLineBeforeStatement{Statements: []string{statement}}
			s := tokens.New(append([]token.Token{}, toks...))
			if changed := f.Fix(s); changed != want {
				t.Errorf("%q with %s: changed=%v, want %v (%q)", v, statement, changed, want, s.Render())
			}
		}
	}
}
