package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen4BExplicitIndirectVariable(t *testing.T) {
	cases := []struct {
		src, want string
		changed   bool
	}{
		// core variable-variable
		{"<?php echo $$foo;", "<?php echo ${$foo};", true},
		{"<?php $$foo = 1;", "<?php ${$foo} = 1;", true},
		// array access stays outside the braces
		{"<?php echo $$foo[0];", "<?php echo ${$foo}[0];", true},
		{"<?php echo $$foo['bar'];", "<?php echo ${$foo}['bar'];", true},
		// property/method access after the indirect variable stays outside
		{"<?php echo $$foo->bar;", "<?php echo ${$foo}->bar;", true},
		// dynamic property access via object operator
		{"<?php echo $foo->$bar;", "<?php echo $foo->{$bar};", true},
		{"<?php echo $foo->$bar['baz'];", "<?php echo $foo->{$bar}['baz'];", true},
		{"<?php echo $foo->$cb($x);", "<?php echo $foo->{$cb}($x);", true},
		// nullsafe operator
		{"<?php echo $foo?->$bar;", "<?php echo $foo?->{$bar};", true},
		// nested variable-variable: only the concrete "$name" is braced
		{"<?php echo $$$foo;", "<?php echo $${$foo};", true},
		// two independent indirect variables in one statement
		{"<?php $$a = $$b;", "<?php ${$a} = ${$b};", true},

		// already explicit - no-op
		{"<?php echo ${$foo};", "<?php echo ${$foo};", false},
		{"<?php echo $foo->{$bar};", "<?php echo $foo->{$bar};", false},
		// a plain variable is untouched
		{"<?php echo $foo;", "<?php echo $foo;", false},
		// a normal property access (non-dynamic) is untouched
		{"<?php echo $foo->bar;", "<?php echo $foo->bar;", false},
		// static/method calls with plain names are untouched
		{"<?php $foo->bar();", "<?php $foo->bar();", false},
	}
	for _, c := range cases {
		got, changed := apply(t, ExplicitIndirectVariable{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		if again, _ := apply(t, ExplicitIndirectVariable{}, got); again != got {
			t.Fatalf("not idempotent: %q -> %q", got, again)
		}
	}
}

func TestGen4BSourceURLs(t *testing.T) {
	fixers := []fixer.Fixer{
		ExplicitIndirectVariable{},
	}
	for _, f := range fixers {
		if want := fixer.SourceURLFor(f.Name()); f.SourceURL() != want {
			t.Fatalf("%T SourceURL()=%q want %q", f, f.SourceURL(), want)
		}
	}
}
