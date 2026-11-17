package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// applyWithIndent runs a fixer on src with a configured indentation level.
func applyWithIndent(t *testing.T, r fixerRule, indent, src string) string {
	t.Helper()
	s := tokens.New(lexer.Lex(src))
	s.SetIndent(indent)
	r.Fix(s)
	return s.Render()
}

// expected outputs come from PHP-CS-Fixer run with WhitespacesFixerConfig("\t", "\n")
func TestIndentationTypeWithTabs(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"<?php\n\t$a = 1;\n\t\t$b = 2;\n":                  "<?php\n\t$a = 1;\n\t\t$b = 2;\n",
		"<?php\n    $a = 1;\n        $b = 2;\n":            "<?php\n\t$a = 1;\n\t\t$b = 2;\n",
		"<?php\nif (1) {\n  \t$a = 1;\n      $b = 2;\n}\n": "<?php\nif (1) {\n\t$a = 1;\n\t  $b = 2;\n}\n",
		"<?php\n/**\n     * x\n \t * y\n\t\t */\nfoo();\n": "<?php\n/**\n\t * x\n\t * y\n\t\t */\nfoo();\n",
		"<?php\n// c\n    $a = 1;\n":                       "<?php\n// c\n\t$a = 1;\n",
	}
	for src, want := range cases {
		if got := applyWithIndent(t, IndentationType{}, "\t", src); got != want {
			t.Errorf("src=%q\ngot  %q\nwant %q", src, got, want)
		}
	}
}
