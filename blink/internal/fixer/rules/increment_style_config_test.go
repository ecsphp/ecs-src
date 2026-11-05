package rules

import "testing"

// The expected output of every case and configuration comes from the PHP engine.
func TestIncrementStyleConfigs(t *testing.T) {
	t.Parallel()
	fixers := []fixerRule{
		IncrementStyle{},
		IncrementStyle{}.WithConfig(map[string]any{"style": "pre"}).(fixerRule),
		IncrementStyle{}.WithConfig(map[string]any{"style": "post"}).(fixerRule),
	}
	cases := []struct {
		src  string
		want [3]string
	}{
		{"<?php\n$i++;\n$i--;\n++$i;\n--$i;\n", [3]string{"<?php\n++$i;\n--$i;\n++$i;\n--$i;\n", "<?php\n++$i;\n--$i;\n++$i;\n--$i;\n", "<?php\n$i++;\n$i--;\n$i++;\n$i--;\n"}},
		{"<?php\n$a->b++;\n++$a->b;\n$a?->b++;\n", [3]string{"<?php\n++$a->b;\n++$a->b;\n++$a?->b;\n", "<?php\n++$a->b;\n++$a->b;\n++$a?->b;\n", "<?php\n$a->b++;\n$a->b++;\n$a?->b++;\n"}},
		{"<?php\n$a[0]++;\n++$a[0];\n$a[$k][1]--;\n--$a[$k][1];\n", [3]string{"<?php\n++$a[0];\n++$a[0];\n--$a[$k][1];\n--$a[$k][1];\n", "<?php\n++$a[0];\n++$a[0];\n--$a[$k][1];\n--$a[$k][1];\n", "<?php\n$a[0]++;\n$a[0]++;\n$a[$k][1]--;\n$a[$k][1]--;\n"}},
		{"<?php\nFoo::$bar++;\n++Foo::$bar;\n\\A\\B::$c--;\n--\\A\\B::$c;\nstatic::$n++;\n++static::$n;\nself::$n++;\n++self::$n;\n", [3]string{"<?php\n++Foo::$bar;\n++Foo::$bar;\n--\\A\\B::$c;\n--\\A\\B::$c;\n++static::$n;\n++static::$n;\n++self::$n;\n++self::$n;\n", "<?php\n++Foo::$bar;\n++Foo::$bar;\n--\\A\\B::$c;\n--\\A\\B::$c;\n++static::$n;\n++static::$n;\n++self::$n;\n++self::$n;\n", "<?php\nFoo::$bar++;\nFoo::$bar++;\n\\A\\B::$c--;\n\\A\\B::$c--;\nstatic::$n++;\nstatic::$n++;\nself::$n++;\nself::$n++;\n"}},
		{"<?php\n$a->b->c[1]++;\n++$a->b->c[1];\n$a::$b->c++;\n++$a::$b->c;\n", [3]string{"<?php\n++$a->b->c[1];\n++$a->b->c[1];\n++$a::$b->c;\n++$a::$b->c;\n", "<?php\n++$a->b->c[1];\n++$a->b->c[1];\n++$a::$b->c;\n++$a::$b->c;\n", "<?php\n$a->b->c[1]++;\n$a->b->c[1]++;\n$a::$b->c++;\n$a::$b->c++;\n"}},
		{"<?php\n${'x'}++;\n++${'x'};\n$$x++;\n++$$x;\n$a->{'b'}++;\n++$a->{'b'};\n$a->$b++;\n++$a->$b;\n", [3]string{"<?php\n++${'x'};\n++${'x'};\n++$$x;\n++$$x;\n++$a->{'b'};\n++$a->{'b'};\n++$a->$b;\n++$a->$b;\n", "<?php\n++${'x'};\n++${'x'};\n++$$x;\n++$$x;\n++$a->{'b'};\n++$a->{'b'};\n++$a->$b;\n++$a->$b;\n", "<?php\n${'x'}++;\n${'x'}++;\n$$x++;\n$$x++;\n$a->{'b'}++;\n$a->{'b'}++;\n$a->$b++;\n$a->$b++;\n"}},
		{"<?php\nfor ($i = 0; $i < 3; $i++) {}\nfor ($i = 0; $i < 3; ++$i) {}\nfor (;; $i++, $j++) {}\nfor (;; ++$i, ++$j) {}\n", [3]string{"<?php\nfor ($i = 0; $i < 3; ++$i) {}\nfor ($i = 0; $i < 3; ++$i) {}\nfor (;; $i++, $j++) {}\nfor (;; ++$i, ++$j) {}\n", "<?php\nfor ($i = 0; $i < 3; ++$i) {}\nfor ($i = 0; $i < 3; ++$i) {}\nfor (;; $i++, $j++) {}\nfor (;; ++$i, ++$j) {}\n", "<?php\nfor ($i = 0; $i < 3; $i++) {}\nfor ($i = 0; $i < 3; $i++) {}\nfor (;; $i++, $j++) {}\nfor (;; ++$i, ++$j) {}\n"}},
		{"<?php\nif ($x) $i++;\nif ($x) ++$i;\nwhile ($x) { $i--; }\nwhile ($x) { --$i; }\n", [3]string{"<?php\nif ($x) ++$i;\nif ($x) ++$i;\nwhile ($x) { --$i; }\nwhile ($x) { --$i; }\n", "<?php\nif ($x) ++$i;\nif ($x) ++$i;\nwhile ($x) { --$i; }\nwhile ($x) { --$i; }\n", "<?php\nif ($x) $i++;\nif ($x) $i++;\nwhile ($x) { $i--; }\nwhile ($x) { $i--; }\n"}},
		{"<?php\n$a = $i++;\n$a = ++$i;\nfoo($i++);\nfoo(++$i);\nfoo($a, $i++);\nreturn $i++;\nreturn ++$i;\n", [3]string{"<?php\n$a = $i++;\n$a = ++$i;\nfoo($i++);\nfoo(++$i);\nfoo($a, $i++);\nreturn $i++;\nreturn ++$i;\n", "<?php\n$a = $i++;\n$a = ++$i;\nfoo($i++);\nfoo(++$i);\nfoo($a, $i++);\nreturn $i++;\nreturn ++$i;\n", "<?php\n$a = $i++;\n$a = ++$i;\nfoo($i++);\nfoo(++$i);\nfoo($a, $i++);\nreturn $i++;\nreturn ++$i;\n"}},
		{"<?php\n$i ++;\n++ $i;\n$i /* c */ ++;\n++ /* c */ $i;\n$i++ ;\n++$i ;\n$i++ /* c */;\n++$i /* c */;\n", [3]string{"<?php\n++$i ;\n++ $i;\n++$i /* c */ ;\n++ /* c */ $i;\n++$i ;\n++$i ;\n++$i /* c */;\n++$i /* c */;\n", "<?php\n++$i ;\n++ $i;\n++$i /* c */ ;\n++ /* c */ $i;\n++$i ;\n++$i ;\n++$i /* c */;\n++$i /* c */;\n", "<?php\n$i ++;\n $i++;\n$i /* c */ ++;\n /* c */ $i++;\n$i++ ;\n$i ++;\n$i++ /* c */;\n$i ++/* c */;\n"}},
		{"<?php $i++; ?>\n<?php ++$i; ?>\n", [3]string{"<?php ++$i; ?>\n<?php ++$i; ?>\n", "<?php ++$i; ?>\n<?php ++$i; ?>\n", "<?php $i++; ?>\n<?php $i++; ?>\n"}},
		{"<?php\n$i++ + 1;\n++$i + 1;\n$a->b()++;\n++$a->b();\n($i++);\n(++$i);\n", [3]string{"<?php\n$i++ + 1;\n++$i + 1;\n$a->b()++;\n++$a->b();\n($i++);\n(++$i);\n", "<?php\n$i++ + 1;\n++$i + 1;\n$a->b()++;\n++$a->b();\n($i++);\n(++$i);\n", "<?php\n$i++ + 1;\n++$i + 1;\n$a->b()++;\n$a->b()++;\n($i++);\n(++$i);\n"}},
		{"<?php\n$a[$i++]++;\n++$a[++$i];\n$a[foo($x)]->b++;\n++$a[foo($x)]->b;\n", [3]string{"<?php\n++$a[$i++];\n++$a[++$i];\n++$a[foo($x)]->b;\n++$a[foo($x)]->b;\n", "<?php\n++$a[$i++];\n++$a[++$i];\n++$a[foo($x)]->b;\n++$a[foo($x)]->b;\n", "<?php\n$a[$i++]++;\n$a[++$i]++;\n$a[foo($x)]->b++;\n$a[foo($x)]->b++;\n"}},
		{"<?php\n$i\n++;\n++\n$i;\n", [3]string{"<?php\n++$i\n;\n++\n$i;\n", "<?php\n++$i\n;\n++\n$i;\n", "<?php\n$i\n++;\n\n$i++;\n"}},
		{"<?php\nfunction f() {\n    $this->count++;\n    ++$this->count;\n    parent::$x++;\n    ++parent::$x;\n}\n", [3]string{"<?php\nfunction f() {\n    ++$this->count;\n    ++$this->count;\n    ++parent::$x;\n    ++parent::$x;\n}\n", "<?php\nfunction f() {\n    ++$this->count;\n    ++$this->count;\n    ++parent::$x;\n    ++parent::$x;\n}\n", "<?php\nfunction f() {\n    $this->count++;\n    $this->count++;\n    parent::$x++;\n    parent::$x++;\n}\n"}},
		{"<?php\n\"$x\"; $i++;\n`$x`; ++$i;\n$a = \"{$b}\"; $i++;\n", [3]string{"<?php\n\"$x\"; ++$i;\n`$x`; ++$i;\n$a = \"{$b}\"; ++$i;\n", "<?php\n\"$x\"; ++$i;\n`$x`; ++$i;\n$a = \"{$b}\"; ++$i;\n", "<?php\n\"$x\"; $i++;\n`$x`; $i++;\n$a = \"{$b}\"; $i++;\n"}},
		{"<?php\n$a?->b->c++;\n++$a?->b->c;\n$a::$b::$c++;\n++$a::$b::$c;\n$a->b::$c++;\n++$a->b::$c;\n", [3]string{"<?php\n++$a?->b->c;\n++$a?->b->c;\n++$a::$b::$c;\n++$a::$b::$c;\n$a->b::$c++;\n++$a->b::$c;\n", "<?php\n++$a?->b->c;\n++$a?->b->c;\n++$a::$b::$c;\n++$a::$b::$c;\n$a->b::$c++;\n++$a->b::$c;\n", "<?php\n$a?->b->c++;\n$a?->b->c++;\n$a::$b::$c++;\n$a::$b::$c++;\n$a->b::$c++;\n$a->b::$c++;\n"}},
		{"<?php\nif ($x) { $i++; } else $j--;\nfor (;;) $i++;\n$i++ ?>\n", [3]string{"<?php\nif ($x) { ++$i; } else $j--;\nfor (;;) ++$i;\n$i++ ?>\n", "<?php\nif ($x) { ++$i; } else $j--;\nfor (;;) ++$i;\n$i++ ?>\n", "<?php\nif ($x) { $i++; } else $j--;\nfor (;;) $i++;\n$i++ ?>\n"}},
		{"<?php\n$a->b[0]->c++;\n++$a->b[0]->c;\n$a->{$k}[0]--;\n--$a->{$k}[0];\n", [3]string{"<?php\n++$a->b[0]->c;\n++$a->b[0]->c;\n--$a->{$k}[0];\n--$a->{$k}[0];\n", "<?php\n++$a->b[0]->c;\n++$a->b[0]->c;\n--$a->{$k}[0];\n--$a->{$k}[0];\n", "<?php\n$a->b[0]->c++;\n$a->b[0]->c++;\n$a->{$k}[0]--;\n$a->{$k}[0]--;\n"}},
		{"<?php\n$x = 1 + $i++;\n$x = [$i++];\n$f = fn() => $i++;\nif ($i++) {}\nif (++$i) {}\n", [3]string{"<?php\n$x = 1 + $i++;\n$x = [$i++];\n$f = fn() => $i++;\nif ($i++) {}\nif (++$i) {}\n", "<?php\n$x = 1 + $i++;\n$x = [$i++];\n$f = fn() => $i++;\nif ($i++) {}\nif (++$i) {}\n", "<?php\n$x = 1 + $i++;\n$x = [$i++];\n$f = fn() => $i++;\nif ($i++) {}\nif (++$i) {}\n"}},
	}
	for _, c := range cases {
		for k, f := range fixers {
			got, changed := apply(t, f, c.src)
			if got != c.want[k] || changed != (c.want[k] != c.src) {
				t.Errorf("config %d src=%q\nchanged=%v got=%q\nwant=%q", k, c.src, changed, got, c.want[k])
			}
			if again, changed := apply(t, f, got); changed {
				t.Errorf("config %d not idempotent on %q: %q", k, got, again)
			}
		}
	}
}
