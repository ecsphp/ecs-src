package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen3FuncFunctionDeclaration(t *testing.T) {
	f := FunctionDeclaration{}
	cases := []struct {
		src, want string
		changed   bool
	}{
		// named function: collapse space after keyword, glue name to "("
		{"<?php function  foo  ($a){}", "<?php function foo($a){}", true},
		// method with modifiers
		{"<?php public function  m ($x){}", "<?php public function m($x){}", true},
		// by-reference: one space after keyword, "&" glued to name, name glued to "("
		{"<?php function  &  foo  (){}", "<?php function &foo(){}", true},
		{"<?php function&foo (){}", "<?php function &foo(){}", true},
		// closure: one space between "function" and "("
		{"<?php $f = function(){};", "<?php $f = function (){};", true},
		{"<?php $f = function  (){};", "<?php $f = function (){};", true},
		// closure with use: one space around "use"
		{"<?php $f = function()use($x){};", "<?php $f = function () use ($x){};", true},
		// static closure: one space after "static" and after "function"
		{"<?php $f = static   function(){};", "<?php $f = static function (){};", true},

		// already correct - no-op
		{"<?php function foo($x){}", "<?php function foo($x){}", false},
		{"<?php public function m($x){}", "<?php public function m($x){}", false},
		{"<?php function &foo(){}", "<?php function &foo(){}", false},
		{"<?php $f = function () use ($x) {};", "<?php $f = function () use ($x) {};", false},
		{"<?php $f = static function () {};", "<?php $f = static function () {};", false},
		// a function call must never be touched
		{"<?php strlen ($x);", "<?php strlen ($x);", false},
		{"<?php foo();", "<?php foo();", false},
		// a "use function" import is not a declaration
		{"<?php use function ns\\f;", "<?php use function ns\\f;", false},
		// arrow fn is left alone (fn spacing is out of scope)
		{"<?php $f = fn($x) => $x * 2;", "<?php $f = fn ($x) => $x * 2;", true},
		{"<?php $f = fn ($x) => $x * 2;", "<?php $f = fn ($x) => $x * 2;", false},
		{"<?php $f = fn () => $x;", "<?php $f = fn () => $x;", false},
		// newline between keyword and name is kept intact
		{"<?php function\nfoo($x){}", "<?php function\nfoo($x){}", false},
	}
	for _, c := range cases {
		got, changed := apply(t, f, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		// idempotent: a second pass is a no-op
		if again, ch2 := apply(t, f, got); again != got || ch2 {
			t.Fatalf("not idempotent: %q -> %q (changed=%v)", got, again, ch2)
		}
	}
}

func TestGen3FuncSourceURL(t *testing.T) {
	f := FunctionDeclaration{}
	if want := fixer.SourceURLFor(f.Name()); f.SourceURL() != want {
		t.Fatalf("%T SourceURL()=%q want %q", f, f.SourceURL(), want)
	}
}
