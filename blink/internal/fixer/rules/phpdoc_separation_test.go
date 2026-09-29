package rules

import "testing"

func TestPhpdocSeparation(t *testing.T) {
	f := PhpdocSeparation{}

	// different unlisted tags get a blank line; same-name tags stay together
	got, changed := apply(t, f, "<?php\n/**\n * @throws Exception foo\n * @param string $foo\n * @param bool $bar\n * @return int\n */\nfunction f($foo, $bar) {}\n")
	want := "<?php\n/**\n * @throws Exception foo\n *\n * @param string $foo\n * @param bool $bar\n *\n * @return int\n */\nfunction f($foo, $bar) {}\n"
	if !changed || got != want {
		t.Fatalf("mixed: changed=%v got=%q", changed, got)
	}

	// tags of the same default group stay together
	if _, changed := apply(t, f, "<?php\n/**\n * @author A\n * @license MIT\n */\nclass A {}\n"); changed {
		t.Fatal("author+license are one group, must stay together")
	}

	// description is separated from the first tag
	got, changed = apply(t, f, "<?php\n/**\n * Hello.\n * @param int $x\n */\nfunction f($x) {}\n")
	want = "<?php\n/**\n * Hello.\n *\n * @param int $x\n */\nfunction f($x) {}\n"
	if !changed || got != want {
		t.Fatalf("description: changed=%v got=%q", changed, got)
	}

	// extra blank lines between same-name tags are collapsed
	got, changed = apply(t, f, "<?php\n/**\n * @param int $x\n *\n *\n * @param int $y\n */\nfunction f($x, $y) {}\n")
	want = "<?php\n/**\n * @param int $x\n * @param int $y\n */\nfunction f($x, $y) {}\n"
	if !changed || got != want {
		t.Fatalf("collapse: changed=%v got=%q", changed, got)
	}
}
