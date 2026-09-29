package rules

import "testing"

func TestExplicitStringVariable(t *testing.T) {
	f := ExplicitStringVariable{}

	cases := []struct {
		src, want string
		changed   bool
	}{
		{`<?php $s = "$a";`, `<?php $s = "{$a}";`, true},
		{`<?php $s = "x $a y";`, `<?php $s = "x {$a} y";`, true},
		{`<?php $s = "$a->b";`, `<?php $s = "{$a->b}";`, true},
		{`<?php $s = "$a->b->c";`, `<?php $s = "{$a->b}->c";`, true}, // one level only
		{`<?php $s = "$a[0]";`, `<?php $s = "{$a[0]}";`, true},
		{`<?php $s = "$a[key]";`, `<?php $s = "{$a['key']}";`, true},    // bareword quoted
		{`<?php $s = "$a[$k]";`, `<?php $s = "{$a[$k]}";`, true},        // var index kept
		{`<?php $s = "{$a}";`, `<?php $s = "{$a}";`, false},             // already explicit
		{`<?php $s = "${a}";`, `<?php $s = "${a}";`, false},             // complex, left alone
		{`<?php $s = 'no $a here';`, `<?php $s = 'no $a here';`, false}, // single-quoted
		{`<?php $s = "esc \$a";`, `<?php $s = "esc \$a";`, false},       // escaped
	}
	for _, c := range cases {
		got, changed := apply(t, f, c.src)
		if got != c.want || changed != c.changed {
			t.Errorf("src=%q\n got=%q changed=%v\nwant=%q changed=%v", c.src, got, changed, c.want, c.changed)
		}
	}

	// heredoc interpolates; nowdoc does not
	got, changed := apply(t, f, "<?php $s = <<<EOT\nhi $a\nEOT;\n")
	if want := "<?php $s = <<<EOT\nhi {$a}\nEOT;\n"; got != want || !changed {
		t.Errorf("heredoc: got=%q changed=%v want=%q", got, changed, want)
	}
	if _, changed := apply(t, f, "<?php $s = <<<'EOT'\nhi $a\nEOT;\n"); changed {
		t.Error("nowdoc must not interpolate")
	}
}
