package rules

import "testing"

func TestMethodChainingNewline(t *testing.T) {
	f := MethodChainingNewline{}

	cases := []struct {
		src, want string
		changed   bool
	}{
		// plain variable-rooted chain with a long trailing call: split
		{"<?php\n$x->first()->second();", "<?php\n$x->first()\n    ->second();", true},

		// short (<=5 chars) no-argument trailing calls stay inline
		{"<?php\n$x->one()->two();", "<?php\n$x->one()->two();", false},
		{"<?php\nreturn $this->a()->b()->c();", "<?php\nreturn $this->a()->b()->c();", false},

		// single method call: nothing to split
		{"<?php\n$x->one();", "<?php\n$x->one();", false},

		// a "::" earlier on the line suppresses the split (symplify heuristic)
		{"<?php\nstatic::$r = $b->one()->second();", "<?php\nstatic::$r = $b->one()->second();", false},
		// a "[" earlier on the line likewise
		{"<?php\n$s = $a->b($m[2])->second();", "<?php\n$s = $a->b($m[2])->second();", false},

		// a chain inline inside a call's arguments is left alone
		{"<?php\nfoo($x->a()->second());", "<?php\nfoo($x->a()->second());", false},
		// grouped root: the first call stays inline, later calls split
		{"<?php\n$y = (new Foo())->bar()->second();", "<?php\n$y = (new Foo())->bar()\n    ->second();", true},
		{"<?php\n$y = (new Foo())->bar();", "<?php\n$y = (new Foo())->bar();", false},

		// a chain used in a boolean or comparison expression stays inline
		{"<?php\n$r = $this->first()->second() || $x;", "<?php\n$r = $this->first()->second() || $x;", false},
		// a chain inside an if condition stays inline
		{"<?php\nif ($this->first()->second()) {\n}", "<?php\nif ($this->first()->second()) {\n}", false},

		// already multi-line: no-op
		{"<?php\n$x->one()\n    ->two();", "<?php\n$x->one()\n    ->two();", false},
	}
	for _, c := range cases {
		got, changed := apply(t, f, c.src)
		if got != c.want || changed != c.changed {
			t.Errorf("src=%q\n got=%q changed=%v\nwant=%q changed=%v", c.src, got, changed, c.want, c.changed)
		}
	}
}
