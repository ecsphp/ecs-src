package rules

import "testing"

func TestOperatorLinebreakConfig(t *testing.T) {
	t.Parallel()

	// position: end moves an operator from the start of a line to the end of the
	// previous one.
	src := "<?php\n$a = $b\n&& $c;"
	f := OperatorLinebreak{}.WithConfig(map[string]any{"position": "end"})
	got, changed := apply(t, f.(fixerRule), src)
	if want := "<?php\n$a = $b &&\n$c;"; !changed || got != want {
		t.Fatalf("position end changed=%v got=%q want=%q", changed, got, want)
	}

	// only_booleans leaves a non-boolean operator (concat) untouched.
	concat := "<?php\n$a = $b .\n$c;"
	f = OperatorLinebreak{}.WithConfig(map[string]any{"only_booleans": true})
	if _, changed := apply(t, f.(fixerRule), concat); changed {
		t.Fatal("only_booleans must leave the concat operator alone")
	}
	// the default still moves the concat operator to the beginning.
	if got, changed := apply(t, OperatorLinebreak{}, concat); !changed || got != "<?php\n$a = $b\n. $c;" {
		t.Fatalf("default concat changed=%v got=%q", changed, got)
	}
}
