package rules

import "testing"

// The expected output of every case comes from the PHP engine.
func TestBracesPositionComment(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"if ($b) // c\n{\n    echo 1;\n}", "if ($b) { // c\necho 1;\n}"},
		{"if ($b) # c\n{\n    echo 1;\n}", "if ($b) { # c\necho 1;\n}"},
		{"if ($b) /* c */\n{\n    echo 1;\n}", "if ($b) { /* c */\necho 1;\n}"},
		{"if ($b) /* c */ {\n    echo 1;\n}", "if ($b) { /* c */\n    echo 1;\n}"},
		{"if ($b)/* c */{\n    echo 1;\n}", "if ($b) {/* c */\n    echo 1;\n}"},
		{"if ($b) /** d */\n{\n    echo 1;\n}", "if ($b) { /** d */\necho 1;\n}"},
		{"if ($b)\n// c\n{\n    echo 1;\n}", "if ($b) {\n// c\necho 1;\n}"},
		{"if ($b) // c\n// e\n{\n    echo 1;\n}", "if ($b) { // c\n// e\necho 1;\n}"},
		{"if ($b) // c\n/* e */\n{\n    echo 1;\n}", "if ($b) { // c\n/* e */\necho 1;\n}"},
		{"if ($b) // c\n\n{\n    echo 1;\n}", "if ($b) { // c\n\necho 1;\n}"},
		{"if ($b)// c\n{\n    echo 1;\n}", "if ($b) {// c\necho 1;\n}"},
		{"if ($b) { // c\n    echo 1;\n}", "if ($b) { // c\n    echo 1;\n}"},
		{"if ($b) // c\n{ // e\n    echo 1;\n}", "if ($b) { // c\n// e\n    echo 1;\n}"},
		{"foreach ($a as $b) // c\n{\n    echo 1;\n}", "foreach ($a as $b) { // c\necho 1;\n}"},
		{"while ($b) // c\n{\n    echo 1;\n}", "while ($b) { // c\necho 1;\n}"},
		{"for (;;) // c\n{\n    echo 1;\n}", "for (;;) { // c\necho 1;\n}"},
		{"do // c\n{\n    echo 1;\n} while ($b);", "do { // c\necho 1;\n} while ($b);"},
		{"try // c\n{\n    echo 1;\n} catch (E $e) // e\n{\n} finally // f\n{\n}", "try { // c\necho 1;\n} catch (E $e) { // e\n} finally { // f\n}"},
		{"switch ($b) // c\n{\n    case 1:\n}", "switch ($b) { // c\ncase 1:\n}"},
		{"if ($b) {\n} else // c\n{\n    echo 1;\n}", "if ($b) {\n} else { // c\necho 1;\n}"},
		{"if ($b) {\n} elseif ($c) // c\n{\n    echo 1;\n}", "if ($b) {\n} elseif ($c) { // c\necho 1;\n}"},
		{"$a = new class // c\n{\n    public $x;\n};", "$a = new class { // c\npublic $x;\n};"},
		{"$a = new class # c\n{\n    public $x;\n};", "$a = new class { # c\npublic $x;\n};"},
		{"$a = new class /* c */\n{\n    public $x;\n};", "$a = new class { /* c */\npublic $x;\n};"},
		{"$a = new class /* c */ {\n    public $x;\n};", "$a = new class { /* c */\n    public $x;\n};"},
		{"$a = new class($b) // c\n{\n    public $x;\n};", "$a = new class($b) { // c\npublic $x;\n};"},
		{"$a = new class($b) implements C // c\n{\n    public $x;\n};", "$a = new class($b) implements C { // c\npublic $x;\n};"},
		{"$a = new class\n// c\n{\n    public $x;\n};", "$a = new class {\n// c\npublic $x;\n};"},
		{"class A // c\n{\n}", "class A // c\n{\n}"},
		{"class A /* c */ {\n}", "class A /* c */\n{\n}"},
		{"class A // c\n{\n    public $x;\n}", "class A // c\n{\n    public $x;\n}"},
		{"class A { // c\n    public $x;\n}", "class A // c\n{public $x;\n}"},
		{"class A { /* c */\n    public $x;\n}", "class A\n{ /* c */\n    public $x;\n}"},
		{"class A { # c\n    public $x;\n}", "class A # c\n{public $x;\n}"},
		{"class A { /** d */\n    public $x;\n}", "class A\n{ /** d */\n    public $x;\n}"},
		{"class A\n{ // c\n    public $x;\n}", "class A\n{ // c\n    public $x;\n}"},
		{"class A {// c\n    public $x;\n}", "class A // c\n{public $x;\n}"},
		{"function f() // c\n{\n}", "function f() // c\n{\n}"},
		{"function f() { // c\n    echo 1;\n}", "function f() // c\n{echo 1;\n}"},
		{"function f(): void // c\n{\n    echo 1;\n}", "function f(): void // c\n{\n    echo 1;\n}"},
		{"function f() /* c */ {\n    echo 1;\n}", "function f() /* c */\n{\n    echo 1;\n}"},
	}
	for _, c := range cases {
		got, changed := apply(t, BracesPosition{}, "<?php\n"+c.src+"\n")
		if want := "<?php\n" + c.want + "\n"; got != want || changed != (c.src != c.want) {
			t.Errorf("%s\n got=%q changed=%v\nwant=%q", c.src, got, changed, want)
		}
	}
}
