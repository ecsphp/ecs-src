package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen5PhpdocOrderTypesOrder(t *testing.T) {
	t.Parallel()
	f := PhpdocTypesOrder{}

	// null moves to the end, other members keep order: string|null|int -> string|int|null
	assertFix(t, f,
		"<?php\n/**\n * @param string|null|int $x\n */\nfunction f($x){}",
		"<?php\n/**\n * @param string|int|null $x\n */\nfunction f($x){}", true)

	// no null member - order is left untouched (sort_algorithm=none)
	assertFix(t, f,
		"<?php\n/**\n * @return string|int\n */\nfunction f(){}",
		"<?php\n/**\n * @return string|int\n */\nfunction f(){}", false)

	// single-line @var: null already last - no-op
	assertFix(t, f,
		"<?php\n/** @var int|null */\npublic $x;",
		"<?php\n/** @var int|null */\npublic $x;", false)

	// null already last - no-op
	assertFix(t, f,
		"<?php\n/**\n * @param int|string $x\n */\nfunction f($x){}",
		"<?php\n/**\n * @param int|string $x\n */\nfunction f($x){}", false)

	// single member - no-op
	assertFix(t, f,
		"<?php\n/**\n * @param int $x\n */\nfunction f($x){}",
		"<?php\n/**\n * @param int $x\n */\nfunction f($x){}", false)

	// generics are skipped (contains '<')
	assertFix(t, f,
		"<?php\n/**\n * @param array<int,string>|null $x\n */\nfunction f($x){}",
		"<?php\n/**\n * @param array<int,string>|null $x\n */\nfunction f($x){}", false)

	// DNF is skipped (contains '(')
	assertFix(t, f,
		"<?php\n/**\n * @param (A&B)|null $x\n */\nfunction f($x){}",
		"<?php\n/**\n * @param (A&B)|null $x\n */\nfunction f($x){}", false)

	// array shape is skipped (contains '{')
	assertFix(t, f,
		"<?php\n/**\n * @var array{a:int}|null\n */\npublic $x;",
		"<?php\n/**\n * @var array{a:int}|null\n */\npublic $x;", false)

	// original spelling of members is preserved; null moved to the end
	assertFix(t, f,
		"<?php\n/**\n * @param \\Foo|null|Bar $x\n */\nfunction f($x){}",
		"<?php\n/**\n * @param \\Foo|Bar|null $x\n */\nfunction f($x){}", true)
}

func TestGen5PhpdocOrderVarAnnotation(t *testing.T) {
	t.Parallel()
	f := PhpdocVarAnnotationCorrectOrder{}

	// swapped single-line @var
	assertFix(t, f,
		"<?php\n/** @var $foo int */\n$foo = 1;",
		"<?php\n/** @var int $foo */\n$foo = 1;", true)

	// swapped multi-line @var
	assertFix(t, f,
		"<?php\n/**\n * @var $foo string\n */\n$foo = 'x';",
		"<?php\n/**\n * @var string $foo\n */\n$foo = 'x';", true)

	// @type tag, generic type
	assertFix(t, f,
		"<?php\n/** @type $foo array<int, string> */\n$foo = [];",
		"<?php\n/** @type array<int, string> $foo */\n$foo = [];", true)

	// already correct - no-op
	assertFix(t, f,
		"<?php\n/** @var int $foo */\n$foo = 1;",
		"<?php\n/** @var int $foo */\n$foo = 1;", false)

	// no type after the variable - no-op
	assertFix(t, f,
		"<?php\n/** @var $foo */\n$foo = 1;",
		"<?php\n/** @var $foo */\n$foo = 1;", false)

	// unrelated tag - no-op
	assertFix(t, f,
		"<?php\n/** @param int $foo */\nfunction f($foo){}",
		"<?php\n/** @param int $foo */\nfunction f($foo){}", false)
}

func TestGen5PhpdocOrderSourceURLs(t *testing.T) {
	t.Parallel()
	for _, f := range []fixer.Fixer{PhpdocTypesOrder{}, PhpdocVarAnnotationCorrectOrder{}} {
		if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
			t.Fatalf("%s SourceURL %q, want %q", f.Name(), got, want)
		}
	}
}
