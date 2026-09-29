package rules

import "testing"

func TestControlStructureContinuationPosition(t *testing.T) {
	got, changed := apply(t, ControlStructureContinuationPosition{}, "<?php\nif ($a) {\n    x();\n}\nelse {\n    y();\n}")
	if want := "<?php\nif ($a) {\n    x();\n} else {\n    y();\n}"; !changed || got != want {
		t.Fatalf("else: changed=%v got=%q want=%q", changed, got, want)
	}
	// try/catch/finally continuations
	got, changed = apply(t, ControlStructureContinuationPosition{}, "<?php\ntry {\n}\ncatch (E $e) {\n}\nfinally {\n}")
	if want := "<?php\ntry {\n} catch (E $e) {\n} finally {\n}"; !changed || got != want {
		t.Fatalf("try: changed=%v got=%q want=%q", changed, got, want)
	}
	// do-while: the trailing while moves up
	got, changed = apply(t, ControlStructureContinuationPosition{}, "<?php\ndo {\n    x();\n}\nwhile ($a);")
	if want := "<?php\ndo {\n    x();\n} while ($a);"; !changed || got != want {
		t.Fatalf("do-while: changed=%v got=%q want=%q", changed, got, want)
	}
	// a plain while loop header is not a continuation
	if _, changed := apply(t, ControlStructureContinuationPosition{}, "<?php\n}\nwhile ($a) {\n}"); changed {
		t.Fatal("loop while must not merge with an unrelated brace")
	}
	// already on the same line: no-op
	if _, changed := apply(t, ControlStructureContinuationPosition{}, "<?php\nif ($a) {\n} else {\n}"); changed {
		t.Fatal("same-line continuation must not change")
	}
}

func TestNoUnneededControlParentheses(t *testing.T) {
	got, changed := apply(t, NoUnneededControlParentheses{}, "<?php return ($x);")
	if want := "<?php return $x;"; !changed || got != want {
		t.Fatalf("return: changed=%v got=%q want=%q", changed, got, want)
	}
	// echo, print, yield, break, continue, clone, case
	cases := map[string]string{
		"<?php echo ($y);":       "<?php echo $y;",
		"<?php print ($y);":      "<?php print $y;",
		"<?php yield ($y);":      "<?php yield $y;",
		"<?php break (2);":       "<?php break 2;",
		"<?php continue (1);":    "<?php continue 1;",
		"<?php $b = clone ($o);": "<?php $b = clone $o;",
	}
	for src, want := range cases {
		if got, changed := apply(t, NoUnneededControlParentheses{}, src); !changed || got != want {
			t.Fatalf("%q: changed=%v got=%q want=%q", src, changed, got, want)
		}
	}
	got, changed = apply(t, NoUnneededControlParentheses{}, "<?php switch ($a) { case ($f): break; }")
	if want := "<?php switch ($a) { case $f: break; }"; !changed || got != want {
		t.Fatalf("case: changed=%v got=%q want=%q", changed, got, want)
	}
	// nested pairs are peeled in one pass
	got, changed = apply(t, NoUnneededControlParentheses{}, "<?php return (($x));")
	if want := "<?php return $x;"; !changed || got != want {
		t.Fatalf("nested: changed=%v got=%q want=%q", changed, got, want)
	}
	// no space between keyword and "(" still yields a valid separator
	got, changed = apply(t, NoUnneededControlParentheses{}, "<?php return($x);")
	if want := "<?php return $x;"; !changed || got != want {
		t.Fatalf("no-space: changed=%v got=%q want=%q", changed, got, want)
	}
	// left alone: partial sub-expression, multiple args, member call, already clean
	for _, src := range []string{
		"<?php return ($a) + $b;",
		"<?php echo ($a), ($b);",
		"<?php return $x;",
		"<?php $r = Foo::case($x);",
	} {
		if _, changed := apply(t, NoUnneededControlParentheses{}, src); changed {
			t.Fatalf("must not change: %q", src)
		}
	}
}
