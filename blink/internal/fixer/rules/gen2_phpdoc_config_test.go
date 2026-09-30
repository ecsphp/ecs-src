package rules

import "testing"

func TestPhpdocOrderByValueConfig(t *testing.T) {
	src := "<?php\n/**\n * @covers B\n * @covers A\n */\n"
	got, changed := apply(t, PhpdocOrderByValue{}, src)
	if want := "<?php\n/**\n * @covers A\n * @covers B\n */\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	// configured to order @author instead: @covers is left alone
	f := PhpdocOrderByValue{}.WithConfig(map[string]any{"annotations": []any{"author"}})
	got, changed = apply(t, f.(fixerRule), "<?php\n/**\n * @author B\n * @author A\n */\n")
	if want := "<?php\n/**\n * @author A\n * @author B\n */\n"; !changed || got != want {
		t.Fatalf("author: changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, f.(fixerRule), src); changed {
		t.Fatal("covers must not be ordered when only author is configured")
	}
}
