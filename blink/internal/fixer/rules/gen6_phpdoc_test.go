package rules

import "testing"

func TestNoAliasLanguageConstructCall(t *testing.T) {
	got, changed := apply(t, NoAliasLanguageConstructCall{}, "<?php die('x'); DIE; Die();")
	if want := "<?php exit('x'); exit; exit();"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// already exit, and member/namespaced names are left alone
	if _, changed := apply(t, NoAliasLanguageConstructCall{}, "<?php exit('x'); $o->die(); Foo::die();"); changed {
		t.Fatal("exit and member die() must not change")
	}
}

func TestBlankLineBeforeStatement(t *testing.T) {
	got, changed := apply(t, BlankLineBeforeStatement{}, "<?php\nfunction f($a)\n{\n    $b = 1;\n    return $b;\n}\n")
	if want := "<?php\nfunction f($a)\n{\n    $b = 1;\n\n    return $b;\n}\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// first statement in a block gets no blank line
	if _, changed := apply(t, BlankLineBeforeStatement{}, "<?php\nfunction f()\n{\n    return 1;\n}\n"); changed {
		t.Fatal("first statement in block must not gain a blank line")
	}
	// already separated: idempotent
	if _, changed := apply(t, BlankLineBeforeStatement{}, "<?php\nfunction f()\n{\n    $b = 1;\n\n    return $b;\n}\n"); changed {
		t.Fatal("existing blank line must not change")
	}
}

func TestPhpdocSummary(t *testing.T) {
	got, changed := apply(t, PhpdocSummary{}, "<?php\n/**\n * Summary without period\n */\nfunction f() {}")
	if want := "<?php\n/**\n * Summary without period.\n */\nfunction f() {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// already punctuated
	if _, changed := apply(t, PhpdocSummary{}, "<?php\n/**\n * Done.\n */\nfunction f() {}"); changed {
		t.Fatal("summary ending in a period must not change")
	}
	// label-style multi-line summary is skipped
	if _, changed := apply(t, PhpdocSummary{}, "<?php\n/**\n * Example:\n * more text\n */\nfunction f() {}"); changed {
		t.Fatal("label summary ending in ':' must not change")
	}
}

func TestPhpdocTagType(t *testing.T) {
	got, changed := apply(t, PhpdocTagType{}, "<?php\n/**\n * {@inheritdoc}\n */\nfunction f() {}")
	if want := "<?php\n/**\n * @inheritdoc\n */\nfunction f() {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// already annotation form
	if _, changed := apply(t, PhpdocTagType{}, "<?php\n/**\n * @inheritDoc\n */\nfunction f() {}"); changed {
		t.Fatal("annotation form must not change")
	}
}

func TestPhpdocOrder(t *testing.T) {
	src := "<?php\n/**\n * @param int $a\n * @return int\n * @throws \\Exception\n */\nfunction f($a) {}"
	got, changed := apply(t, PhpdocOrder{}, src)
	want := "<?php\n/**\n * @param int $a\n * @throws \\Exception\n * @return int\n */\nfunction f($a) {}"
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// already ordered
	if _, changed := apply(t, PhpdocOrder{}, want); changed {
		t.Fatal("already-ordered tags must not change")
	}
}
