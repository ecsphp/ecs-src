package rules

import "testing"

// The expected output of every case comes from the PHP engine.
func TestMethodArgumentSpaceCommentGap(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"function f(\n    string $a,      // first\n    ?int $b = null,   // second\n    bool $c = true,\n): void {\n}", "function f(\n    string $a,      // first\n    ?int $b = null,   // second\n    bool $c = true,\n): void {\n}"},
		{"function f(\n    ?int $a = null,       /* third */\n    bool $b = true,\n): void {\n}", "function f(\n    ?int $a = null,       /* third */\n    bool $b = true,\n): void {\n}"},
		{"foo(\n    $a,     // aligned\n    $bb,    // aligned\n    $ccc\n);", "foo(\n    $a,     // aligned\n    $bb,    // aligned\n    $ccc\n);"},
		{"foo(\n    $a, // one space\n    $b\n);", "foo(\n    $a, // one space\n    $b\n);"},
		{"foo(\n    $a,// no space\n    $b\n);", "foo(\n    $a,// no space\n    $b\n);"},
		{"foo(\n    $a,   /* block */ $b\n);", "foo(\n    $a, /* block */\n    $b\n);"},
		{"foo($a,   $b);", "foo($a, $b);"},
		{"foo(\n    $a,   \n    $b\n);", "foo(\n    $a,\n    $b\n);"},
		{"foo($a,   // c\n    $b);", "foo($a,   // c\n    $b);"},
	}
	for _, c := range cases {
		got, changed := apply(t, MethodArgumentSpace{}, "<?php\n"+c.src+"\n")
		if want := "<?php\n" + c.want + "\n"; got != want || changed != (c.src != c.want) {
			t.Errorf("%s\n got=%q changed=%v\nwant=%q", c.src, got, changed, want)
		}
	}
}
