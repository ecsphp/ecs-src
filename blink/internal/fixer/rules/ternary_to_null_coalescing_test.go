package rules

import "testing"

func TestTernaryToNullCoalescing(t *testing.T) {
	f := TernaryToNullCoalescing{}

	cases := []struct {
		src, want string
		changed   bool
	}{
		{`<?php $a = isset($x) ? $x : null;`, `<?php $a = $x ?? null;`, true},
		{`<?php $b = isset($arr[1]['k']) ? $arr[1]['k'] : 'd';`, `<?php $b = $arr[1]['k'] ?? 'd';`, true},
		{`<?php $c = isset($o->p) ? $o->p : 5;`, `<?php $c = $o->p ?? 5;`, true},
		{`<?php $d = isset($x, $y) ? $x : $y;`, `<?php $d = isset($x, $y) ? $x : $y;`, false},      // multi-arg
		{`<?php $e = isset($x) ? $y : $z;`, `<?php $e = isset($x) ? $y : $z;`, false},              // mismatch
		{`<?php $g = isset($x) ? $x : (isset($y) ? $y : 0);`, `<?php $g = $x ?? ($y ?? 0);`, true}, // nested isset also converts
	}
	for _, c := range cases {
		got, changed := apply(t, f, c.src)
		if got != c.want || changed != c.changed {
			t.Errorf("src=%q\n got=%q changed=%v\nwant=%q changed=%v", c.src, got, changed, c.want, c.changed)
		}
	}
}
