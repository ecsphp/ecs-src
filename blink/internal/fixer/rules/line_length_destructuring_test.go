package rules

import "testing"

// Symplify's LineLengthFixer only processes array literals, so an over-long
// destructuring target is left alone while an array literal is still broken.
// Expectations are the PHP fixer's own output.
func TestLineLengthSkipsDestructuring(t *testing.T) {
	t.Parallel()
	const items = "$aaaaaaaaaaaaaaaaaaaa, $bbbbbbbbbbbbbbbbbbbbbbbb, $cccccccccccccccccccccccc, $dddddddddddddddddddddd, $eeeeeeeeeeee"
	for name, src := range map[string]string{
		"assignment":  "<?php\n[" + items + "] = foo();\n",
		"foreach":     "<?php\nforeach ($rows as [" + items + "]) {\n}\n",
		"foreach key": "<?php\nforeach ($rows as $key => [" + items + "]) {\n}\n",
		"nested":      "<?php\n[[" + items + "], $f] = foo();\n",
	} {
		if got, changed := apply(t, LineLength{}, src); changed {
			t.Errorf("%s must not change, got %q", name, got)
		}
	}

	got, _ := apply(t, LineLength{}, "<?php\n$x = ["+items+"];\n")
	want := "<?php\n$x = [\n    $aaaaaaaaaaaaaaaaaaaa,\n    $bbbbbbbbbbbbbbbbbbbbbbbb,\n    $cccccccccccccccccccccccc,\n    $dddddddddddddddddddddd,\n    $eeeeeeeeeeee\n];\n"
	if got != want {
		t.Errorf("array literal:\n got %q\nwant %q", got, want)
	}
}
