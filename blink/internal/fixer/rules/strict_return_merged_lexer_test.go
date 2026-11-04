package rules

import "testing"

// Inputs that depend on blink's lexer giving token_get_all's tokens: reserved
// words as named-argument labels, an uppercase open tag, "yield from" with a
// comment inside. The expected outputs come from the PHP fixers.
func TestStrictReturnRulesOnMergedLexer(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		f       fixerRule
		in, out string
	}{
		{"no useless else, return label", NoUselessElse{}, "<?php if ($a) { foo(return: 1); } else { bar(); }", "<?php if ($a) { foo(return: 1); } else { bar(); }"},
		{"no useless else, exit label", NoUselessElse{}, "<?php if ($a) { foo(exit: 1); } else { bar(); }", "<?php if ($a) { foo(exit: 1); } else { bar(); }"},
		{"no useless else, function label", NoUselessElse{}, "<?php if ($a) { return; } else { foo(function: 1); }", "<?php if ($a) { return; }   foo(function: 1); "},
		{"no superfluous elseif, exit label", NoSuperfluousElseif{}, "<?php if ($a) { foo(exit: 1); } elseif ($b) { bar(); }", "<?php if ($a) { foo(exit: 1); } elseif ($b) { bar(); }"},
		{"declare strict types, uppercase open tag", DeclareStrictTypes{}, "<?PHP\nfoo();\n", "<?PHP declare(strict_types=1);\nfoo();\n"},
		{"void return, yield from with a comment", VoidReturn{}, "<?php function f() { yield /* c */ from g(); }", "<?php function f() { yield /* c */ from g(); }"},
		{"static lambda, array callable call", StaticLambda{}, "<?php $a = [$o, \"m\"](function () {})->call($b);", "<?php $a = [$o, \"m\"](function () {})->call($b);"},
	} {
		if got, _ := apply(t, c.f, c.in); got != c.out {
			t.Errorf("%s:\ngot  %q\nwant %q", c.name, got, c.out)
		}
	}
}
