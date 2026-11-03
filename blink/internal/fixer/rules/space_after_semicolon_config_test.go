package rules

import "testing"

func TestSpaceAfterSemicolonConfigCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"default keeps empty for", SpaceAfterSemicolon{}, "<?php for ($i = 0; ; ++$i) {}", "<?php for ($i = 0; ; ++$i) {}"},
		{"remove in empty for", SpaceAfterSemicolon{}.WithConfig(map[string]any{"remove_in_empty_for_expressions": true}).(fixerRule), "<?php for ($i = 0; ; ++$i) {}", "<?php for ($i = 0;; ++$i) {}"},
		{"remove keeps normal statements", SpaceAfterSemicolon{}.WithConfig(map[string]any{"remove_in_empty_for_expressions": true}).(fixerRule), "<?php $a=1;$b=2;", "<?php $a=1; $b=2;"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}

// The same input through both settings of remove_in_empty_for_expressions,
// with PHP-CS-Fixer's output for each, including its quirk of keeping the
// space before ")" once a space was inserted earlier in the same for header.
func TestSpaceAfterSemicolonMatchesPhpCsFixer(t *testing.T) {
	t.Parallel()
	src := "<?php\nfor ($i=0;$i<1; ) {}\nfor ($i = 0; $i < 1; ) {}\nfor (;;) {}\nfor ( ; ; ) {}\n" +
		"for ($i = 1;; ++$i) {}\n$a = 1;;\n$b = 1;   $c = 2;\n$d = 1;\t$e = 2;\n$f = 1;  // c\n" +
		"$g = 1;\t/* c */\nif (1) { $h = 1;  }\n?>\n<?php echo 1;?>\n<?php echo 2;"
	wantRemove := "<?php\nfor ($i=0; $i<1; ) {}\nfor ($i = 0; $i < 1;) {}\nfor (;;) {}\nfor ( ;;) {}\n" +
		"for ($i = 1;; ++$i) {}\n$a = 1;;\n$b = 1; $c = 2;\n$d = 1; $e = 2;\n$f = 1;  // c\n" +
		"$g = 1;\t/* c */\nif (1) { $h = 1; }\n?>\n<?php echo 1; ?>\n<?php echo 2;"
	wantDefault := "<?php\nfor ($i=0; $i<1; ) {}\nfor ($i = 0; $i < 1; ) {}\nfor (; ;) {}\nfor ( ; ; ) {}\n" +
		"for ($i = 1; ; ++$i) {}\n$a = 1; ;\n$b = 1; $c = 2;\n$d = 1; $e = 2;\n$f = 1;  // c\n" +
		"$g = 1;\t/* c */\nif (1) { $h = 1; }\n?>\n<?php echo 1; ?>\n<?php echo 2;"
	remove := SpaceAfterSemicolon{}.WithConfig(map[string]any{"remove_in_empty_for_expressions": true}).(fixerRule)
	// not idempotent, like the PHP fixer: a second pass drops that space too
	if got, changed := apply(t, remove, src); !changed || got != wantRemove {
		t.Fatalf("remove_in_empty_for_expressions: changed=%v\n got=%q\nwant=%q", changed, got, wantRemove)
	}
	assertFix(t, SpaceAfterSemicolon{}, src, wantDefault, true)
}
