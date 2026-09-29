package rules

import "testing"

func TestGenWsEncoding(t *testing.T) {
	got, changed := apply(t, Encoding{}, "\uFEFF<?php echo 1;\n")
	if want := "<?php echo 1;\n"; !changed || got != want {
		t.Fatalf("strip BOM: changed=%v got=%q want=%q", changed, got, want)
	}
	// BOM in front of inline HTML is trimmed, HTML preserved
	got, changed = apply(t, Encoding{}, "\uFEFF<html><?php echo 1;")
	if want := "<html><?php echo 1;"; !changed || got != want {
		t.Fatalf("html BOM: changed=%v got=%q want=%q", changed, got, want)
	}
	// clean file untouched
	if _, changed := apply(t, Encoding{}, "<?php echo 1;\n"); changed {
		t.Fatal("no BOM must not change")
	}
	idempotent(t, Encoding{}, "<?php echo 1;\n")
}

func TestGenWsDeclareParentheses(t *testing.T) {
	got, changed := apply(t, DeclareParentheses{}, "<?php declare ( strict_types = 1 );")
	if want := "<?php declare(strict_types = 1);"; !changed || got != want {
		t.Fatalf("spaces: changed=%v got=%q want=%q", changed, got, want)
	}
	got, changed = apply(t, DeclareParentheses{}, "<?php declare\n(\n    strict_types=1\n);")
	if want := "<?php declare(strict_types=1);"; !changed || got != want {
		t.Fatalf("newlines: changed=%v got=%q want=%q", changed, got, want)
	}
	// already tight
	if _, changed := apply(t, DeclareParentheses{}, "<?php declare(strict_types=1);"); changed {
		t.Fatal("tight declare must not change")
	}
	// a method named declare is not the construct
	if _, changed := apply(t, DeclareParentheses{}, "<?php $o->declare ( 1 );"); changed {
		t.Fatal("method call declare() must not change")
	}
	idempotent(t, DeclareParentheses{}, "<?php declare(strict_types = 1);")
}

func TestGenWsMultilineCommentOpeningClosing(t *testing.T) {
	cases := []struct{ in, want string }{
		{"<?php /*** Opening comment */", "<?php /* Opening comment */"},
		{"<?php /** Closing DocBlock ***/", "<?php /** Closing DocBlock */"},
		{"<?php /* Closing comment ***/", "<?php /* Closing comment */"},
		{"<?php /***/", "<?php /**/"},
		{"<?php /********/", "<?php /**/"},
	}
	for _, c := range cases {
		got, changed := apply(t, MultilineCommentOpeningClosing{}, c.in)
		if !changed || got != c.want {
			t.Fatalf("in=%q changed=%v got=%q want=%q", c.in, changed, got, c.want)
		}
	}
	// genuine doc block and false doc block are left alone
	noop := []string{
		"<?php /** Opening DocBlock */",
		"<?php /*\\ Opening false-DocBlock */",
		"<?php // /*** not a block comment",
		"<?php /**/",
	}
	for _, src := range noop {
		if _, changed := apply(t, MultilineCommentOpeningClosing{}, src); changed {
			t.Fatalf("must not change: %q", src)
		}
	}
	idempotent(t, MultilineCommentOpeningClosing{}, "<?php /* x */")
}
