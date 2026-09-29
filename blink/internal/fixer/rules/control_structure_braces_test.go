package rules

import "testing"

func TestControlStructureBraces(t *testing.T) {
	cases := []struct {
		src     string
		want    string
		changed bool
	}{
		// braceless if body gets wrapped
		{`<?php if ($x) echo 'a';`, `<?php if ($x) { echo 'a'; }`, true},
		// else + elseif chain
		{`<?php if ($x) foo(); elseif ($y) bar(); else baz();`, `<?php if ($x) { foo(); } elseif ($y) { bar(); } else { baz(); }`, true},
		// braceless while
		{`<?php while ($x) $x--;`, `<?php while ($x) { $x--; }`, true},
		// braceless for
		{`<?php for ($i=0;$i<3;$i++) run();`, `<?php for ($i=0;$i<3;$i++) { run(); }`, true},
		// braceless foreach
		{`<?php foreach ($a as $b) use_it($b);`, `<?php foreach ($a as $b) { use_it($b); }`, true},
		// nested braceless if
		{`<?php if ($x) if ($y) go();`, `<?php if ($x) { if ($y) { go(); } }`, true},
		// braceless do/while
		{`<?php do run(); while ($x);`, `<?php do { run(); } while ($x);`, true},

		// already braced: no-op
		{`<?php if ($x) { echo 'a'; }`, `<?php if ($x) { echo 'a'; }`, false},
		{"<?php if ($x) {\n    foo();\n} else {\n    bar();\n}", "<?php if ($x) {\n    foo();\n} else {\n    bar();\n}", false},
		{`<?php foreach ($a as $b) { c($b); }`, `<?php foreach ($a as $b) { c($b); }`, false},
		// alternative syntax left untouched (NoAlternativeSyntax handles it)
		{`<?php if ($x): echo 'a'; endif;`, `<?php if ($x): echo 'a'; endif;`, false},
		// method named like a control keyword is not a control
		{`<?php $o->for($a);`, `<?php $o->for($a);`, false},
		// try/catch/finally already braced
		{`<?php try { a(); } catch (E $e) { b(); } finally { c(); }`, `<?php try { a(); } catch (E $e) { b(); } finally { c(); }`, false},
		// a constant named like a control keyword is an identifier, not a control
		{`<?php class C { const IF = 1; }`, `<?php class C { const IF = 1; }`, false},
		{`<?php class C { public const string IF = __DIR__; }`, `<?php class C { public const string IF = __DIR__; }`, false},
		{`<?php class C { const FOR = 1; const WHILE = 2; }`, `<?php class C { const FOR = 1; const WHILE = 2; }`, false},
	}
	for _, c := range cases {
		got, changed := apply(t, ControlStructureBraces{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		if again, changed2 := apply(t, ControlStructureBraces{}, got); changed2 || again != got {
			t.Fatalf("not idempotent: %q -> %q changed=%v", got, again, changed2)
		}
	}
}
