package rules

import "testing"

func TestNoMixedEchoPrint(t *testing.T) {
	got, changed := apply(t, NoMixedEchoPrint{}, `<?php print "a";`)
	if want := `<?php echo "a";`; !changed || got != want {
		t.Fatalf("statement: changed=%v got=%q want=%q", changed, got, want)
	}
	// print as an expression is left alone
	if _, changed := apply(t, NoMixedEchoPrint{}, `<?php $x = print "c";`); changed {
		t.Fatal("print as expression must not change")
	}
	if _, changed := apply(t, NoMixedEchoPrint{}, `<?php foo(print "d");`); changed {
		t.Fatal("print as argument must not change")
	}
	// after ")" and "else" it is a statement
	got, changed = apply(t, NoMixedEchoPrint{}, `<?php if ($y) print "e"; else print "f";`)
	if want := `<?php if ($y) echo "e"; else echo "f";`; !changed || got != want {
		t.Fatalf("branch: changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, NoMixedEchoPrint{}, `<?php echo "g";`); changed {
		t.Fatal("echo must not change")
	}
}

func TestNoAliasFunctions(t *testing.T) {
	got, changed := apply(t, NoAliasFunctions{}, `<?php sizeof($a); join($x); is_double($n);`)
	if want := `<?php count($a); implode($x); is_float($n);`; !changed || got != want {
		t.Fatalf("aliases: changed=%v got=%q want=%q", changed, got, want)
	}
	// a fully-qualified global call is still rewritten
	got, changed = apply(t, NoAliasFunctions{}, `<?php \sizeof($a);`)
	if want := `<?php \count($a);`; !changed || got != want {
		t.Fatalf("fqn: changed=%v got=%q want=%q", changed, got, want)
	}
	// methods and namespaced names are not global functions
	if _, changed := apply(t, NoAliasFunctions{}, `<?php $o->sizeof($a); Foo\join($x); Bar::join($y);`); changed {
		t.Fatal("method/namespaced/static must not change")
	}
	if _, changed := apply(t, NoAliasFunctions{}, `<?php count($a);`); changed {
		t.Fatal("canonical name must not change")
	}
}

func TestMultilineWhitespaceBeforeSemicolons(t *testing.T) {
	got, changed := apply(t, MultilineWhitespaceBeforeSemicolons{}, "<?php $a = foo()\n    ;")
	if want := "<?php $a = foo();"; !changed || got != want {
		t.Fatalf("plain: changed=%v got=%q want=%q", changed, got, want)
	}
	// a comment before the ";" keeps the ";" with the code, ahead of the comment
	got, changed = apply(t, MultilineWhitespaceBeforeSemicolons{}, "<?php $a = foo() // c\n;")
	if want := "<?php $a = foo(); // c"; !changed || got != want {
		t.Fatalf("comment: changed=%v got=%q want=%q", changed, got, want)
	}
	// a const statement's own ";" is left alone
	if _, changed := apply(t, MultilineWhitespaceBeforeSemicolons{}, "<?php const A = 1;"); changed {
		t.Fatal("const semicolon must not change")
	}
	// single-line ";" is untouched
	if _, changed := apply(t, MultilineWhitespaceBeforeSemicolons{}, "<?php $a = 1;"); changed {
		t.Fatal("single-line semicolon must not change")
	}
}

func TestSingleLineEmptyBodyRegistered(t *testing.T) {
	// smoke test that the collapse runs (the fixer already had unit coverage)
	got, changed := apply(t, SingleLineEmptyBody{}, "<?php class A\n{\n}")
	if want := "<?php class A {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}
