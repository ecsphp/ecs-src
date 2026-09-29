package rules

import "testing"

func TestNoAlternativeSyntax(t *testing.T) {
	cases := []struct {
		src     string
		want    string
		changed bool
	}{
		// if / else / endif
		{`<?php if (true): echo 't'; else: echo 'f'; endif;`, `<?php if (true) { echo 't'; } else { echo 'f'; }`, true},
		// if / elseif / else / endif (elseif brace has no leading space, matching PHP-CS-Fixer)
		{`<?php if ($a): x(); elseif ($b): y(); else: z(); endif;`, `<?php if ($a) { x(); } elseif ($b){ y(); } else { z(); }`, true},
		// foreach / endforeach
		{`<?php foreach ($a as $b): echo $b; endforeach;`, `<?php foreach ($a as $b) { echo $b; }`, true},
		// while / endwhile
		{`<?php while ($x): $x--; endwhile;`, `<?php while ($x) { $x--; }`, true},
		// for / endfor
		{`<?php for ($i=0;$i<3;$i++): run(); endfor;`, `<?php for ($i=0;$i<3;$i++) { run(); }`, true},
		// switch / endswitch
		{`<?php switch ($x): case 1: a(); endswitch;`, `<?php switch ($x) { case 1: a(); }`, true},
		// tight colon: braces still get their conditional spaces
		{`<?php if(true):echo 't';endif;`, `<?php if(true) { echo 't';}`, true},

		// already braces: no-op
		{`<?php if (true) { echo 't'; } else { echo 'f'; }`, `<?php if (true) { echo 't'; } else { echo 'f'; }`, false},
		// ternary colon is not alternative syntax
		{`<?php $x = $a ? 1 : 2;`, `<?php $x = $a ? 1 : 2;`, false},
		// switch case colon without endswitch stays
		{`<?php switch ($x) { case 1: a(); }`, `<?php switch ($x) { case 1: a(); }`, false},
		// inline HTML block is converted too (fix_non_monolithic_code=true default)
		{"<?php if ($x): ?>hi<?php endif; ?>", "<?php if ($x) { ?>hi<?php } ?>", true},
	}
	for _, c := range cases {
		got, changed := apply(t, NoAlternativeSyntax{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		if again, changed2 := apply(t, NoAlternativeSyntax{}, got); changed2 || again != got {
			t.Fatalf("not idempotent: %q -> %q changed=%v", got, again, changed2)
		}
	}
}
