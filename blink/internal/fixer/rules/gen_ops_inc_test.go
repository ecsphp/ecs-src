package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGenOpsIncStandardizeIncrement(t *testing.T) {
	cases := []struct {
		src, want string
		changed   bool
	}{
		{"<?php $i += 1;", "<?php ++$i;", true},
		{"<?php $i -= 1;", "<?php --$i;", true},
		{"<?php $i+=1;", "<?php ++$i;", true},
		{"<?php foo($i += 1);", "<?php foo(++$i);", true},
		{"<?php $a = [$i += 1, 2];", "<?php $a = [++$i, 2];", true},
		// already pre-increment: no-op
		{"<?php ++$i;", "<?php ++$i;", false},
		// not the literal 1
		{"<?php $i += 2;", "<?php $i += 2;", false},
		// chained right side must not be touched (precedence)
		{"<?php $i += 1 + 2;", "<?php $i += 1 + 2;", false},
		// complex lvalue is out of scope
		{"<?php $obj->x += 1;", "<?php $obj->x += 1;", false},
		{"<?php $arr[0] += 1;", "<?php $arr[0] += 1;", false},
	}
	for _, c := range cases {
		got, changed := apply(t, StandardizeIncrement{}, c.src)
		if got != c.want || changed != c.changed {
			t.Errorf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
	}
}

func TestGenOpsIncIncrementStyle(t *testing.T) {
	cases := []struct {
		src, want string
		changed   bool
	}{
		{"<?php $i++;", "<?php ++$i;", true},
		{"<?php $i--;", "<?php --$i;", true},
		{"<?php $i ++;", "<?php ++$i;", true},
		{"<?php { $i++; }", "<?php { ++$i; }", true},
		{"<?php $a = 1;\n$i++;", "<?php $a = 1;\n++$i;", true},
		// already pre-increment: no-op
		{"<?php ++$i;", "<?php ++$i;", false},
		// return value used -> not a standalone statement, must not move
		{"<?php $a = $i++;", "<?php $a = $i++;", false},
		{"<?php echo $i++;", "<?php echo $i++;", false},
		{"<?php foo($i++);", "<?php foo($i++);", false},
		// member/offset operand out of scope
		{"<?php $o->x++;", "<?php $o->x++;", false},
		{"<?php $arr[0]++;", "<?php $arr[0]++;", false},
	}
	for _, c := range cases {
		got, changed := apply(t, IncrementStyle{}, c.src)
		if got != c.want || changed != c.changed {
			t.Errorf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
	}
}

func TestGenOpsIncLongToShorthand(t *testing.T) {
	cases := []struct {
		src, want string
		changed   bool
	}{
		{"<?php $a = $a + 1;", "<?php $a += 1;", true},
		{"<?php $a = $a - 1;", "<?php $a -= 1;", true},
		{"<?php $a = $a * $b;", "<?php $a *= $b;", true},
		{"<?php $s = $s . $x;", "<?php $s .= $x;", true},
		{"<?php $a=$a+1;", "<?php $a+=1;", true},
		// already shorthand: no-op
		{"<?php $a += 1;", "<?php $a += 1;", false},
		// different variable on the right
		{"<?php $a = $b + 1;", "<?php $a = $b + 1;", false},
		// chained right side (precedence risk) must not collapse
		{"<?php $a = $a - $b - $c;", "<?php $a = $a - $b - $c;", false},
		{"<?php $a = $a + $b * $c;", "<?php $a = $a + $b * $c;", false},
		// right operand is not self-contained
		{"<?php $a = $a + foo();", "<?php $a = $a + foo();", false},
		{"<?php $a = $a + $b[0];", "<?php $a = $a + $b[0];", false},
		// comparison, not assignment
		{"<?php $x = $a == $a + 1;", "<?php $x = $a == $a + 1;", false},
	}
	for _, c := range cases {
		got, changed := apply(t, LongToShorthandOperator{}, c.src)
		if got != c.want || changed != c.changed {
			t.Errorf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
	}
}

// TestGenOpsIncSourceURLs pins each fixer's SourceURL to the canonical form
// derived from its Name. The three blob URLs were verified to return HTTP 200.
func TestGenOpsIncSourceURLs(t *testing.T) {
	fixers := []fixer.Fixer{
		StandardizeIncrement{},
		IncrementStyle{},
		LongToShorthandOperator{},
	}
	for _, f := range fixers {
		if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
			t.Errorf("%s: SourceURL=%q want=%q", f.Name(), got, want)
		}
	}
}

// TestGenOpsIncIdempotent guarantees a second pass is a no-op for each fixer.
func TestGenOpsIncIdempotent(t *testing.T) {
	fixers := []fixer.Fixer{
		StandardizeIncrement{},
		IncrementStyle{},
		LongToShorthandOperator{},
	}
	corpus := []string{
		"<?php $i += 1;",
		"<?php $i -= 1;",
		"<?php $i++;\n$j--;\n",
		"<?php $a = $a + 1;\n$b = $b . $c;\n",
		"<?php $a = $i++;\n$o->x += 1;\n$a = $a - $b - $c;\n",
	}
	for _, f := range fixers {
		for _, src := range corpus {
			once, _ := apply(t, f, src)
			twice, changed := apply(t, f, once)
			if changed || once != twice {
				t.Errorf("%s not idempotent: src=%q once=%q twice=%q", f.Name(), src, once, twice)
			}
		}
	}
}
