package rules

import "testing"

// The expected output of every case comes from the PHP engine.
func TestBracesPositionCommentNextLine(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"if ($b) { // c\n    echo 1;\n}", "if ($b) // c\n{echo 1;\n}"},
		{"if ($b) { # c\n    echo 1;\n}", "if ($b) # c\n{echo 1;\n}"},
		{"if ($b) { /* c */\n    echo 1;\n}", "if ($b)\n{ /* c */\n    echo 1;\n}"},
		{"if ($b)\n{ // c\n    echo 1;\n}", "if ($b)\n{ // c\n    echo 1;\n}"},
		{"foreach ($a as $b) { // c\n    echo 1;\n}", "foreach ($a as $b) // c\n{echo 1;\n}"},
		{"try { // c\n    echo 1;\n} catch (E $e) { // e\n    echo 2;\n}", "try // c\n{echo 1;\n} catch (E $e) // e\n{echo 2;\n}"},
		{"$a = new class { // c\n    public $x;\n};", "$a = new class // c\n{public $x;\n};"},
		{"$a = new class($b) implements C { // c\n    public $x;\n};", "$a = new class($b) implements C // c\n{public $x;\n};"},
		{"$a = new class /* c */ {\n    public $x;\n};", "$a = new class /* c */\n{\n    public $x;\n};"},
		{"$f = function () { // c\n    echo 1;\n};", "$f = function () // c\n{echo 1;\n};"},
		{"$f = function () use ($a) { // c\n    echo 1;\n};", "$f = function () use ($a) // c\n{echo 1;\n};"},
		{"$f = function () /* c */ {\n    echo 1;\n};", "$f = function () /* c */\n{\n    echo 1;\n};"},
		{"if ($b) // c\n{\n    echo 1;\n}", "if ($b) // c\n{\n    echo 1;\n}"},
	}
	for _, c := range cases {
		got, changed := apply(t, BracesPosition{}.WithConfig(map[string]any{"control_structures_opening_brace": bracesNextLine, "anonymous_classes_opening_brace": bracesNextLine, "anonymous_functions_opening_brace": bracesNextLine}).(fixerRule), "<?php\n"+c.src+"\n")
		if want := "<?php\n" + c.want + "\n"; got != want || changed != (c.src != c.want) {
			t.Errorf("%s\n got=%q changed=%v\nwant=%q", c.src, got, changed, want)
		}
	}
}
