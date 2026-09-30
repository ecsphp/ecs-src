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
}
