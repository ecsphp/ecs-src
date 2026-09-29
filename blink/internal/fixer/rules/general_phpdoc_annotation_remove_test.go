package rules

import "testing"

func TestGeneralPhpdocAnnotationRemove(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		want    string
		changed bool
	}{
		{
			"removes author package group category, keeps others",
			"<?php\n/**\n * Desc.\n *\n * @author Foo\n * @package Bar\n * @group X\n * @category Y\n * @param int $a\n * @see Other\n */\nfunction f($a) {}",
			"<?php\n/**\n * Desc.\n *\n * @param int $a\n * @see Other\n */\nfunction f($a) {}",
			true,
		},
		{
			"no configured tags present is a no-op",
			"<?php\n/**\n * @param int $a\n * @return void\n */\nfunction f($a) {}",
			"<?php\n/**\n * @param int $a\n * @return void\n */\nfunction f($a) {}",
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed := apply(t, GeneralPhpdocAnnotationRemove{}, tc.src)
			if got != tc.want || changed != tc.changed {
				t.Fatalf("changed=%v got=%q want=%q", changed, got, tc.want)
			}
			again, changedAgain := apply(t, GeneralPhpdocAnnotationRemove{}, got)
			if changedAgain || again != got {
				t.Fatalf("not idempotent: changed=%v got=%q", changedAgain, again)
			}
		})
	}
}
