package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen2CasingClassReferenceNameCasing(t *testing.T) {
	cases := []struct {
		src, want string
		changed   bool
	}{
		// fully-qualified built-in class references are recased
		{"<?php new \\exception();", "<?php new \\Exception();", true},
		{"<?php throw new \\runtimeexception('x');", "<?php throw new \\RuntimeException('x');", true},
		{"<?php catch (\\exception $e) {}", "<?php catch (\\Exception $e) {}", true},
		{"<?php function f(\\datetime $d): \\datetimeimmutable {}", "<?php function f(\\DateTime $d): \\DateTimeImmutable {}", true},
		{"<?php $x = \\arrayobject::class;", "<?php $x = \\ArrayObject::class;", true},
		{"<?php $x instanceof \\stdclass;", "<?php $x instanceof \\stdclass;", false}, // ";" guard - left alone
		// same reference without the trailing ";" is recased
		{"<?php if ($x instanceof \\stdclass) {}", "<?php if ($x instanceof \\stdClass) {}", true},
		// already correct - no-op
		{"<?php new \\Exception();", "<?php new \\Exception();", false},
		// not fully qualified - unqualified name may resolve to a local class
		{"<?php new exception();", "<?php new exception();", false},
		// namespaced name "Foo\exception" is not the global built-in
		{"<?php new \\Foo\\exception();", "<?php new \\Foo\\exception();", false},
		// name heads a deeper namespace - not the class itself
		{"<?php new \\exception\\Foo();", "<?php new \\exception\\Foo();", false},
		// member access must never be recased
		{"<?php $o->exception;", "<?php $o->exception;", false},
		{"<?php Foo::exception;", "<?php Foo::exception;", false},
		// unknown / non-built-in names are untouched
		{"<?php new \\MyException();", "<?php new \\MyException();", false},
		{"<?php echo \\PHP_EOL;", "<?php echo \\PHP_EOL;", false},
		// a fully-qualified function call of the same spelling is left alone by
		// the "(" guard (not preceded by new)
		{"<?php \\exception();", "<?php \\exception();", false},
	}
	for _, c := range cases {
		got, changed := apply(t, ClassReferenceNameCasing{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		// idempotent
		if again, _ := apply(t, ClassReferenceNameCasing{}, got); again != got {
			t.Fatalf("not idempotent: %q -> %q", got, again)
		}
	}
}

func TestGen2CasingSourceURLs(t *testing.T) {
	fixers := []fixer.Fixer{
		ClassReferenceNameCasing{},
	}
	for _, f := range fixers {
		if want := fixer.SourceURLFor(f.Name()); f.SourceURL() != want {
			t.Fatalf("%T SourceURL()=%q want %q", f, f.SourceURL(), want)
		}
	}
}
