package rules

import (
	"encoding/json"
	"testing"

	"blink/internal/fixer"
	"blink/internal/lexer"
	"blink/internal/tokens"
)

type fixerRule interface {
	Fix(*tokens.Stream) bool
}

// configuredFromJSON configures f from a config JSON object the way the ECS
// loader does: the raw JSON for a RawConfigurableFixer, else the decoded map.
func configuredFromJSON(t *testing.T, f fixer.ConfigurableFixer, raw string) fixerRule {
	t.Helper()
	if rf, ok := f.(fixer.RawConfigurableFixer); ok {
		return rf.WithRawConfig([]byte(raw)).(fixerRule)
	}
	var config map[string]any
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		t.Fatal(err)
	}
	return f.WithConfig(config).(fixerRule)
}

func apply(t *testing.T, r fixerRule, src string) (string, bool) {
	t.Helper()
	s := tokens.New(lexer.Lex(src))
	changed := r.Fix(s)
	return s.Render(), changed
}

func TestNoSinglelineWhitespaceBeforeSemicolons(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoSinglelineWhitespaceBeforeSemicolons{}, "<?php $x = 1 ; $y = 2  ;")
	if want := "<?php $x = 1; $y = 2;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, NoSinglelineWhitespaceBeforeSemicolons{}, "<?php $x = 1\n;"); changed {
		t.Fatal("multi-line whitespace before ; should be kept")
	}
}

func TestSpaceAfterSemicolon(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, SpaceAfterSemicolon{}, "<?php $a=1;$b=2;")
	if want := "<?php $a=1; $b=2;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// a ";" followed by ")" is left alone, but one followed by ";" (empty for
	// condition) still gains a space, matching php-cs-fixer
	if got, changed := apply(t, SpaceAfterSemicolon{}, "<?php for(;;){}"); !changed || got != "<?php for(; ;){}" {
		t.Fatalf("empty for head: changed=%v got=%q", changed, got)
	}
}

func TestNoWhitespaceInBlankLine(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoWhitespaceInBlankLine{}, "<?php\n$a = 1;\n   \n$b = 2;\n")
	if want := "<?php\n$a = 1;\n\n$b = 2;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestBlankLineAfterOpeningTag(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, BlankLineAfterOpeningTag{}, "<?php\n$a = 1;\n")
	if want := "<?php\n\n$a = 1;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, BlankLineAfterOpeningTag{}, "<?php\n\n$a = 1;\n"); changed {
		t.Fatal("existing blank line must not change")
	}
	// the cases below mirror PHP-CS-Fixer's output for each input
	cases := map[string]string{
		// code on the tag line moves below a blank line
		"<?php $a = 1;\n$b = 2;\n":             "<?php\n\n$a = 1;\n$b = 2;\n",
		"<?php /* c */\n$a = 1;\n":             "<?php\n\n/* c */\n$a = 1;\n",
		"<?php\t$a = 1;\n":                     "<?php\n\n$a = 1;\n",
		"<?php  $a = 1;\n":                     "<?php\n\n $a = 1;\n",
		"<?php echo 1; // x\n":                 "<?php\n\necho 1; // x\n",
		"<?php\n    $a = 1;\n":                 "<?php\n\n    $a = 1;\n",
		"<?php\r\n$a = 1;\r\n":                 "<?php\r\n\n$a = 1;\r\n",
		"#!/usr/bin/env php\n<?php\n$a = 1;\n": "#!/usr/bin/env php\n<?php\n\n$a = 1;\n",
		// a close tag's newline is part of the tag: still monolithic
		"<?php\n$a = 1;\n?>\n": "<?php\n\n$a = 1;\n?>\n",
		// only whitespace after the tag
		"<?php \n":  "<?php\n\n",
		"<?php\n  ": "<?php\n\n  ",
	}
	for src, want := range cases {
		if got, changed := apply(t, BlankLineAfterOpeningTag{}, src); !changed || got != want {
			t.Fatalf("%q: changed=%v got=%q want=%q", src, changed, got, want)
		}
	}
	for _, src := range []string{
		"<?php $a = 1;",                      // no newline at all
		"<?php\n \n$a = 1;\n",                // the token after the tag holds a newline
		"<?php\n$a = 1;\n?>\n<p>x</p>\n",     // inline HTML
		"<?php\n$a = 1; ?>\n<?php $b = 2;\n", // two open tags
		"<?php\n$a = 1;\n?>\n<?= $a ?>\n",    // short echo tag
		"<?php\n",                            // nothing after the tag
		"<?php\r\n",
		"#!/usr/bin/env php\n<?php\n",
	} {
		if got, changed := apply(t, BlankLineAfterOpeningTag{}, src); changed {
			t.Fatalf("%q must not change: got=%q", src, got)
		}
	}
	// only a monolithic PHP file: the data after __halt_compiler is inline HTML,
	// the newline right after "?>" is not
	if _, changed := apply(t, BlankLineAfterOpeningTag{}, "<?php\n$a = 1;\n__halt_compiler();\ndata"); changed {
		t.Fatal("file with data after __halt_compiler must not change")
	}
	got, changed = apply(t, BlankLineAfterOpeningTag{}, "<?php\n$a = 1;\n?>\n")
	if want := "<?php\n\n$a = 1;\n?>\n"; !changed || got != want {
		t.Fatalf("close tag: changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestNoLeadingNamespaceWhitespace(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoLeadingNamespaceWhitespace{}, "<?php\n    namespace App;\n")
	if want := "<?php\nnamespace App;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestBinaryOperatorSpaces(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, BinaryOperatorSpaces{}, "<?php $a = ['x'=>1, 'y'  =>  2];")
	if want := "<?php $a = ['x' => 1, 'y' => 2];"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, BinaryOperatorSpaces{}, "<?php $a = ['x' => 1];"); changed {
		t.Fatal("already-normalized arrow should not change")
	}
	// arrow spanning a newline is left alone (alignment)
	if _, changed := apply(t, BinaryOperatorSpaces{}, "<?php $a = [\n    'x' =>\n    1,\n];"); changed {
		t.Fatal("multi-line arrow should be kept")
	}
	// assignment "=" is normalized too
	got, changed = apply(t, BinaryOperatorSpaces{}, "<?php $a=1; $b  =  2;")
	if want := "<?php $a = 1; $b = 2;"; !changed || got != want {
		t.Fatalf("assign: changed=%v got=%q want=%q", changed, got, want)
	}
	// reference assignment is left alone
	if _, changed := apply(t, BinaryOperatorSpaces{}, "<?php $a = &$b;"); changed {
		t.Fatal("reference assignment should be kept")
	}
}

func TestConcatSpace(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, ConcatSpace{}, "<?php $s = 'a'.'b'. $c;")
	if want := "<?php $s = 'a' . 'b' . $c;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestCastSpaces(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, CastSpaces{}, "<?php $x = (int)$y;")
	if want := "<?php $x = (int) $y;"; !changed || got != want {
		t.Fatalf("simple: changed=%v got=%q want=%q", changed, got, want)
	}
	got, changed = apply(t, CastSpaces{}, "<?php $x = ( string )$y;")
	if want := "<?php $x = (string) $y;"; !changed || got != want {
		t.Fatalf("inner: changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, CastSpaces{}, "<?php $x = (int) $y;"); changed {
		t.Fatal("already-normalized cast should not change")
	}
	// not a cast: function signature and grouping must be untouched
	if _, changed := apply(t, CastSpaces{}, "<?php function f(int $x) {}"); changed {
		t.Fatal("type hint in signature is not a cast")
	}
	if _, changed := apply(t, CastSpaces{}, "<?php $x = ($y);"); changed {
		t.Fatal("grouping parens are not a cast")
	}
}

func TestLowercaseKeywords(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, LowercaseKeywords{}, "<?php FUNCTION foo() { RETURN 1; }")
	if want := "<?php function foo() { return 1; }"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// an enum case name spelled like a keyword must not be lowercased
	if _, changed := apply(t, LowercaseKeywords{}, "<?php enum E { case Default; case Match = 1; }"); changed {
		t.Fatal("enum case name must keep its case")
	}
}

func TestConstantCase(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, ConstantCase{}, "<?php $a = TRUE; $b = NULL; $c = False;")
	if want := "<?php $a = true; $b = null; $c = false;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, ConstantCase{}, "<?php $x->TRUE;"); changed {
		t.Fatal("member named TRUE should not be touched")
	}
}

func TestLowercaseStaticReference(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, LowercaseStaticReference{}, "<?php SELF::x(); PARENT::y(); Static::z();")
	if want := "<?php self::x(); parent::y(); static::z();"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestLowercaseAndShortCast(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, LowercaseCast{}, "<?php $x = (INT)$y;")
	if want := "<?php $x = (int)$y;"; !changed || got != want {
		t.Fatalf("lowercase: changed=%v got=%q want=%q", changed, got, want)
	}
	got, changed = apply(t, ShortScalarCast{}, "<?php $x = (integer)$y; $z = (boolean)$w;")
	if want := "<?php $x = (int)$y; $z = (bool)$w;"; !changed || got != want {
		t.Fatalf("short: changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestSingleSpaceAroundConstruct(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, SingleSpaceAroundConstruct{}, "<?php if($a){} else{}")
	if want := "<?php if ($a){} else {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// static:: is a reference, not a construct body
	if _, changed := apply(t, SingleSpaceAroundConstruct{}, "<?php static::foo();"); changed {
		t.Fatal("static:: should not gain a space")
	}
	// "new static(" / "new class(" are class references, not constructs: no space
	if _, changed := apply(t, SingleSpaceAroundConstruct{}, "<?php return new static($x);"); changed {
		t.Fatal("new static( should not gain a space")
	}
	if got, changed := apply(t, SingleSpaceAroundConstruct{}, "<?php return new static ($x);"); !changed || got != "<?php return new static($x);" {
		t.Fatalf("new static (: changed=%v got=%q", changed, got)
	}
	if _, changed := apply(t, SingleSpaceAroundConstruct{}, "<?php $o = new class($a) {};"); changed {
		t.Fatal("new class( should not gain a space")
	}
	// the "Enum" of "new Enum (" is a name, left to NoSpacesAfterFunctionName
	withEnum := SingleSpaceAroundConstruct{}.WithConfig(map[string]any{"constructs_followed_by_a_single_space": []any{"new", "enum"}}).(fixerRule)
	if _, changed := apply(t, withEnum, "<?php $r = new Enum($type);"); changed {
		t.Fatal("new Enum( should not gain a space")
	}
	if _, changed := apply(t, withEnum, "<?php $r = new Enum ($type);"); changed {
		t.Fatal("new Enum ( is not a construct")
	}
	if got, changed := apply(t, NoSpacesAfterFunctionName{}, "<?php $r = new Enum ($type);"); !changed || got != "<?php $r = new Enum($type);" {
		t.Fatalf("new Enum ( in NoSpacesAfterFunctionName: changed=%v got=%q", changed, got)
	}
}

func TestNoSpacesAfterFunctionName(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoSpacesAfterFunctionName{}, "<?php foo (1); $o->bar ();")
	if want := "<?php foo(1); $o->bar();"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, NoSpacesAfterFunctionName{}, "<?php if ($a) {}"); changed {
		t.Fatal("control keyword is not a function name")
	}
	// "from" in "yield from (expr)" is part of the construct, not a call
	if _, changed := apply(t, NoSpacesAfterFunctionName{}, "<?php function g() { yield from (new Foo())->all(); }"); changed {
		t.Fatal("yield from ( must keep its space")
	}
	// language constructs and a variable call, mirroring php-cs-fixer
	constructs := "<?php\nrequire ('a.php');\nrequire_once ('a.php');\ninclude ('a.php');\ninclude_once ('a.php');\nexit (1);\n$a = isset ($b);\n$a = empty ($b);\nunset ($b);\neval ('1;');\nprint ('x');\necho ('x');\n$c = array (1);\nlist ($d) = [1];\n$func ();\nstrlen ('x');"
	wantConstructs := "<?php\nrequire('a.php');\nrequire_once('a.php');\ninclude('a.php');\ninclude_once('a.php');\nexit(1);\n$a = isset($b);\n$a = empty($b);\nunset($b);\neval('1;');\nprint('x');\necho('x');\n$c = array(1);\nlist($d) = [1];\n$func();\nstrlen('x');"
	if got, changed := apply(t, NoSpacesAfterFunctionName{}, constructs); !changed || got != wantConstructs {
		t.Fatalf("constructs: changed=%v got=%q", changed, got)
	}
	// echo/print/include/require whose parens are expression grouping keep the space
	if _, changed := apply(t, NoSpacesAfterFunctionName{}, "<?php echo ('x') ? 'a' : 'b';"); changed {
		t.Fatal("echo grouping parens must keep their space")
	}
	// a cast is one token in PHP, not a call target: "(int) ($x)" keeps its space,
	// also after a chain of casts and with padding inside the cast
	casts := "<?php $a = (int) ($b + 1); $c = (string) (int) ($d); $e = ( bool ) ($f); $g = (array) ($h ?? []);"
	if got, changed := apply(t, NoSpacesAfterFunctionName{}, casts); changed {
		t.Fatalf("casts: got=%q", got)
	}
	// a parenthesised callee and a dynamic variable are call targets; the whole
	// whitespace goes, newlines included, as in php-cs-fixer
	got, changed = apply(t, NoSpacesAfterFunctionName{}, "<?php ($f) (1); ($g)\n(2); foo\n(3); ${'h'} (4); $o->{'m'} (5);")
	if want := "<?php ($f)(1); ($g)(2); foo(3); ${'h'}(4); $o->{'m'} (5);"; !changed || got != want {
		t.Fatalf("callees: changed=%v got=%q want=%q", changed, got, want)
	}
	// a comment between the callee and "(" stops the fixer, which looks at the
	// previous non-whitespace token: the call is never folded into the comment
	comments := "<?php\nfoo // c\n(1);\nfoo # c\n(2);\nfoo /* c */ (3);\nbar /* c */\n(4);\nfoo /** d */ (5);\n"
	if got, changed := apply(t, NoSpacesAfterFunctionName{}, comments); changed {
		t.Fatalf("comments: got=%q", got)
	}
	// PHP-CS-Fixer gives the "]" of an array literal and the ")" of "(new Foo)"
	// their own token kinds, which are not call targets; an index "]" and a
	// parenthesis before "(new Foo)" are
	closers := "<?php\n$a = [$o, 'm'] (1);\n$b = (new Foo) (2);\n$c = $arr[0] (3);\n$d = foo() [0] (4);\n$e = f(1) (new Foo) (5);\n$f = (new A) (new B) (6);\nif (1) {} (new Foo) (7);\n"
	wantClosers := "<?php\n$a = [$o, 'm'] (1);\n$b = (new Foo) (2);\n$c = $arr[0](3);\n$d = foo() [0](4);\n$e = f(1)(new Foo)(5);\n$f = (new A) (new B)(6);\nif (1) {} (new Foo) (7);\n"
	if got, _ := apply(t, NoSpacesAfterFunctionName{}, closers); got != wantClosers {
		t.Fatalf("closers: got=%q want=%q", got, wantClosers)
	}
}

func TestNoSpacesInsideParenthesis(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoSpacesInsideParenthesis{}, "<?php foo( $a, $b );")
	if want := "<?php foo($a, $b);"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestUnaryOperatorSpaces(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, UnaryOperatorSpaces{}, "<?php $i ++; -- $j;")
	if want := "<?php $i++; --$j;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestElseif(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, Elseif{}, "<?php if ($a) {} else if ($b) {}")
	if want := "<?php if ($a) {} elseif ($b) {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestNoLeadingImportSlash(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoLeadingImportSlash{}, "<?php use \\Foo\\Bar; use function \\ns\\f;")
	if want := "<?php use Foo\\Bar; use function ns\\f;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// a trait use inside a class keeps its leading backslash (not a namespace import)
	if _, changed := apply(t, NoLeadingImportSlash{}, "<?php class C {\n    use \\Foo\\Bar;\n}"); changed {
		t.Fatal("trait use must keep its leading backslash")
	}
}

func TestDeclareEqualNormalize(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, DeclareEqualNormalize{}, "<?php declare(strict_types = 1);")
	if want := "<?php declare(strict_types=1);"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

// TestAllFixersIdempotent guarantees a second --fix pass is a no-op: running
// every fixer twice must equal running it once.
func TestAllFixersIdempotent(t *testing.T) {
	t.Parallel()
	corpus := []string{
		"<?php\ndeclare(strict_types = 1);\nnamespace App;\nuse A\\B, C\\D;\nCLASS Demo\n{\n\n\tpublic function run( $a )\n\t{\n\t\tif($a===TRUE){\n\t\t\treturn SELF::make ( $a );\n\t\t} else if($a) {\n\t\t\t$i ++;\n\t\t}\n\t\t$x=(INTEGER)$a;\n\t\treturn $a.'x';\n\t}\n}\n",
		"<?php $x = <<<EOT\nline $a<$b if($c)\nEOT;\n",
		"<?php $m=['a'=>1,'b'  =>  2];\n",
		"<?php class A { use TraitB, TraitC; }\n",
		"<?php echo 1;\n\n?>\n",
		"<?php\nnamespace App;\n",
		"<?xml version=\"1.0\"?>\n<root><?php echo 1; ?></root>",
		"<?php class A {\n    public function run() {\n        if ($x)\n        {\n            return 1;\n        }\n    }\n}\n",
		"<?php class A {\n    use TraitB, TraitC;\n    function run() {}\n    var $old;\n    const X = 1;\n}\n",
		"<?php\nuse const C\\Z;\nuse B\\Y;\nuse function F\\g;\nuse A\\X;\n\nclass A\n{\n    public int $a, $b;\n}\n",
		"<?php\n$a = 1; $b = 2;$c = 3;\nswitch ($a) { case 1 : break; case 2; break; }\n",
		"<?php\nclass A\n{\nfunction run($a)\n{\nif ($a) {\nreturn 1;\n}\n}\n}\n",
		"<?php\n\nuse App\\Enum\\Action;\n\nfunction foo(\n    int $a,\n    int $b\n): void {\n}\n",
		"<?php\n\nfunction f(?int $a): ?string\n{\n    return $a ? 'y' : 'n';\n}\n",
		"<?php\n\nclass A\n{\n    public const string ARRAY = 'x';\n\n    public function match($x): bool\n    {\n        return $x;\n    }\n}\n",
		"<?php\n\n$a = array(1 ,2, \"plain\");\nif ($a <> $b) {\n};\necho __line__;\n",
		"<?php\n\n/**\n *\n * Summary.\n *\n * @param integer $a\n * @return void\n */\nfunction f($a)\n{\n}\n",
	}
	for _, src := range corpus {
		once := runAll(src)
		twice := runAll(once)
		if once != twice {
			t.Errorf("not idempotent\n src:  %q\n once: %q\n twice:%q", src, once, twice)
		}
	}
}

func runAll(src string) string {
	s := tokens.New(lexer.Lex(src))
	for _, f := range All() {
		f.Fix(s)
	}
	return s.Render()
}

func TestBinaryOperatorSpacesComparison(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, BinaryOperatorSpaces{}, "<?php $x = $a===$b || $c<$d ?? $e;")
	if want := "<?php $x = $a === $b || $c < $d ?? $e;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestBlankLinesBeforeNamespace(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, BlankLinesBeforeNamespace{}, "<?php declare(strict_types=1);\nnamespace App;")
	if want := "<?php declare(strict_types=1);\n\nnamespace App;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestSingleImportPerStatement(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, SingleImportPerStatement{}, "<?php\nuse A\\B, C\\D;")
	if want := "<?php\nuse A\\B;\nuse D;"; got == want {
		t.Fatalf("unexpected: %q", got) // guard against accidental truncation
	}
	if want := "<?php\nuse A\\B;\nuse C\\D;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// closure use is left alone
	if _, changed := apply(t, SingleImportPerStatement{}, "<?php $f = function () use ($a, $b) {};"); changed {
		t.Fatal("closure use must not be split")
	}
	// group imports are expanded by default (group_to_single_imports: true)
	got, changed = apply(t, SingleImportPerStatement{}, "<?php use A\\{B, C};")
	if want := "<?php use A\\B;\nuse A\\C;"; !changed || got != want {
		t.Fatalf("group default changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestNoBlankLinesAfterClassOpening(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoBlankLinesAfterClassOpening{}, "<?php class A\n{\n\n\n    public $x;\n}")
	if want := "<?php class A\n{\n    public $x;\n}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestIndentationType(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, IndentationType{}, "<?php\n\t$a = 1;\n\t\t$b = 2;\n")
	if want := "<?php\n    $a = 1;\n        $b = 2;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestFullOpeningTag(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, FullOpeningTag{}, "<? echo 1;")
	if want := "<?php echo 1;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	got, changed = apply(t, FullOpeningTag{}, "<?$x;")
	if want := "<?php $x;"; !changed || got != want {
		t.Fatalf("separator: changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, FullOpeningTag{}, "<?= $x;"); changed {
		t.Fatal("short echo tag must be left alone")
	}
	if _, changed := apply(t, FullOpeningTag{}, "<?xml version=\"1.0\"?>"); changed {
		t.Fatal("XML declaration must not be converted")
	}
	got, changed = apply(t, FullOpeningTag{}, "<?PHP\nfoo();\n")
	if want := "<?php\nfoo();\n"; !changed || got != want {
		t.Fatalf("uppercase: changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestNoClosingTag(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoClosingTag{}, "<?php echo 1;\n?>\n")
	if want := "<?php echo 1;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// templating file (inline HTML) keeps the closing tag
	if _, changed := apply(t, NoClosingTag{}, "<?php if ($a): ?>\n<div></div>\n<?php endif; ?>\n"); changed {
		t.Fatal("file with inline HTML must keep closing tags")
	}
	// several PHP blocks separated by whitespace-only inline HTML keep the tag
	if _, changed := apply(t, NoClosingTag{}, "<?php echo $a; ?> <?php echo $b; ?>"); changed {
		t.Fatal("multiple PHP blocks must keep the closing tag")
	}
}

func TestBlankLineAfterNamespace(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, BlankLineAfterNamespace{}, "<?php\nnamespace App;\nclass A {}")
	if want := "<?php\nnamespace App;\n\nclass A {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, BlankLineAfterNamespace{}, "<?php\nnamespace App { }"); changed {
		t.Fatal("bracketed namespace must be left alone")
	}
}

func TestSingleLineAfterImports(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, SingleLineAfterImports{}, "<?php\nuse A;\nuse B;\nclass C {}")
	if want := "<?php\nuse A;\nuse B;\n\nclass C {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}

	// inside a braced namespace the following line keeps its indentation
	got, changed = apply(t, SingleLineAfterImports{}, "<?php\nnamespace N {\n    use A;\n    class C {}\n}")
	if want := "<?php\nnamespace N {\n    use A;\n\n    class C {}\n}"; !changed || got != want {
		t.Fatalf("braced: changed=%v got=%q want=%q", changed, got, want)
	}

	// a blank line that already exists is left untouched
	if _, changed := apply(t, SingleLineAfterImports{}, "<?php\nnamespace N {\n    use A;\n\n    class C {}\n}"); changed {
		t.Fatal("existing blank line after import must be a no-op")
	}
}

func TestNoTrailingWhitespace(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoTrailingWhitespace{}, "<?php $x = 1;   \n$y = 2;\t\n")
	if want := "<?php $x = 1;\n$y = 2;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, NoTrailingWhitespace{}, "<?php $x = 1;\n"); changed {
		t.Fatal("clean input should not change")
	}
}

func TestSingleBlankLineAtEndOfFile(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, SingleBlankLineAtEndOfFile{}, "<?php echo 1;\n\n\n")
	if !changed || got != "<?php echo 1;\n" {
		t.Fatalf("collapse: changed=%v got=%q", changed, got)
	}
	got, changed = apply(t, SingleBlankLineAtEndOfFile{}, "<?php echo 1;")
	if !changed || got != "<?php echo 1;\n" {
		t.Fatalf("append: changed=%v got=%q", changed, got)
	}
	if _, changed := apply(t, SingleBlankLineAtEndOfFile{}, "<?php echo 1;\n"); changed {
		t.Fatal("single newline should not change")
	}
	// a file ending in inline HTML gets no trailing newline added (php-cs-fixer
	// skips T_INLINE_HTML / T_CLOSE_TAG endings)
	if _, changed := apply(t, SingleBlankLineAtEndOfFile{}, "<?php echo 1; ?>\n</html>"); changed {
		t.Fatal("inline-HTML ending must be left alone")
	}
	if _, changed := apply(t, SingleBlankLineAtEndOfFile{}, "<?php echo 1; ?>"); changed {
		t.Fatal("close-tag ending must be left alone")
	}
}
