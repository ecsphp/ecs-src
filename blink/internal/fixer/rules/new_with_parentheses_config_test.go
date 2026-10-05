package rules

import "testing"

func TestNewWithParenthesesConfig(t *testing.T) {
	src := "<?php $a = new Foo; $b = new class {};"
	got, _ := apply(t, NewWithParentheses{}, src)
	if want := "<?php $a = new Foo(); $b = new class() {};"; got != want {
		t.Fatalf("default got=%q want=%q", got, want)
	}
	f := NewWithParentheses{}.WithConfig(map[string]any{"named_class": false, "anonymous_class": true})
	got, _ = apply(t, f.(fixerRule), src)
	if want := "<?php $a = new Foo; $b = new class() {};"; got != want {
		t.Fatalf("named off got=%q want=%q", got, want)
	}
	f = NewWithParentheses{}.WithConfig(map[string]any{"anonymous_class": false})
	got, _ = apply(t, f.(fixerRule), src)
	if want := "<?php $a = new Foo(); $b = new class {};"; got != want {
		t.Fatalf("anonymous off got=%q want=%q", got, want)
	}

	// a dynamic class name via "::member" / "[subscript]" already carries its
	// call parentheses, so none are inserted after the leading name
	for _, c := range []string{
		"<?php $x = new static::$builder($query);",
		"<?php $x = new self::$map[0]($a);",
		"<?php $x = new static::$resolvedCollectionClasses[static::class]($models);",
	} {
		if g, changed := apply(t, NewWithParentheses{}, c); changed || g != c {
			t.Fatalf("dynamic class name must be left alone: changed=%v got=%q", changed, g)
		}
	}
}
