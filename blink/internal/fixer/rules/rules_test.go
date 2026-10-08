package rules

import (
	"testing"

	"blink/internal/lexer"
	"blink/internal/tokens"
)

type fixerRule interface {
	Fix(*tokens.Stream) bool
}

func apply(t *testing.T, r fixerRule, src string) (string, bool) {
	t.Helper()
	s := tokens.New(lexer.Lex(src))
	changed := r.Fix(s)
	return s.Render(), changed
}

func TestNoSinglelineWhitespaceBeforeSemicolons(t *testing.T) {
	got, changed := apply(t, NoSinglelineWhitespaceBeforeSemicolons{}, "<?php $x = 1 ; $y = 2  ;")
	if want := "<?php $x = 1; $y = 2;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, NoSinglelineWhitespaceBeforeSemicolons{}, "<?php $x = 1\n;"); changed {
		t.Fatal("multi-line whitespace before ; should be kept")
	}
}

func TestSpaceAfterSemicolon(t *testing.T) {
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
	got, changed := apply(t, NoWhitespaceInBlankLine{}, "<?php\n$a = 1;\n   \n$b = 2;\n")
	if want := "<?php\n$a = 1;\n\n$b = 2;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestBlankLineAfterOpeningTag(t *testing.T) {
	got, changed := apply(t, BlankLineAfterOpeningTag{}, "<?php\n$a = 1;\n")
	if want := "<?php\n\n$a = 1;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, BlankLineAfterOpeningTag{}, "<?php\n\n$a = 1;\n"); changed {
		t.Fatal("existing blank line must not change")
	}
	if _, changed := apply(t, BlankLineAfterOpeningTag{}, "<?php $a = 1;\n"); changed {
		t.Fatal("code on tag line must not change")
	}
}

func TestNoLeadingNamespaceWhitespace(t *testing.T) {
	got, changed := apply(t, NoLeadingNamespaceWhitespace{}, "<?php\n    namespace App;\n")
	if want := "<?php\nnamespace App;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestBinaryOperatorSpaces(t *testing.T) {
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
	got, changed := apply(t, ConcatSpace{}, "<?php $s = 'a'.'b'. $c;")
	if want := "<?php $s = 'a' . 'b' . $c;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestCastSpaces(t *testing.T) {
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
	got, changed := apply(t, LowercaseKeywords{}, "<?php FUNCTION foo() { RETURN 1; }")
	if want := "<?php function foo() { return 1; }"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestConstantCase(t *testing.T) {
	got, changed := apply(t, ConstantCase{}, "<?php $a = TRUE; $b = NULL; $c = False;")
	if want := "<?php $a = true; $b = null; $c = false;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, ConstantCase{}, "<?php $x->TRUE;"); changed {
		t.Fatal("member named TRUE should not be touched")
	}
}

func TestLowercaseStaticReference(t *testing.T) {
	got, changed := apply(t, LowercaseStaticReference{}, "<?php SELF::x(); PARENT::y(); Static::z();")
	if want := "<?php self::x(); parent::y(); static::z();"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestLowercaseAndShortCast(t *testing.T) {
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
	// "new Enum(" is a class reference; "enum" never opens a "(" as a construct,
	// even when "enum" is in the followed list (as ECS configures it)
	withEnum := SingleSpaceAroundConstruct{}.WithConfig(map[string]any{"constructs_followed_by_a_single_space": []any{"new", "enum"}}).(fixerRule)
	if _, changed := apply(t, withEnum, "<?php $r = new Enum($type);"); changed {
		t.Fatal("new Enum( should not gain a space")
	}
	if got, changed := apply(t, withEnum, "<?php $r = new Enum ($type);"); !changed || got != "<?php $r = new Enum($type);" {
		t.Fatalf("new Enum (: changed=%v got=%q", changed, got)
	}
}

func TestNoSpacesAfterFunctionName(t *testing.T) {
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
}

func TestNoSpacesInsideParenthesis(t *testing.T) {
	got, changed := apply(t, NoSpacesInsideParenthesis{}, "<?php foo( $a, $b );")
	if want := "<?php foo($a, $b);"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestUnaryOperatorSpaces(t *testing.T) {
	got, changed := apply(t, UnaryOperatorSpaces{}, "<?php $i ++; -- $j;")
	if want := "<?php $i++; --$j;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestElseif(t *testing.T) {
	got, changed := apply(t, Elseif{}, "<?php if ($a) {} else if ($b) {}")
	if want := "<?php if ($a) {} elseif ($b) {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestNoLeadingImportSlash(t *testing.T) {
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
	got, changed := apply(t, DeclareEqualNormalize{}, "<?php declare(strict_types = 1);")
	if want := "<?php declare(strict_types=1);"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

// TestAllFixersIdempotent guarantees a second --fix pass is a no-op: running
// every fixer twice must equal running it once.
func TestAllFixersIdempotent(t *testing.T) {
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
	got, changed := apply(t, BinaryOperatorSpaces{}, "<?php $x = $a===$b || $c<$d ?? $e;")
	if want := "<?php $x = $a === $b || $c < $d ?? $e;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestBlankLinesBeforeNamespace(t *testing.T) {
	got, changed := apply(t, BlankLinesBeforeNamespace{}, "<?php declare(strict_types=1);\nnamespace App;")
	if want := "<?php declare(strict_types=1);\n\nnamespace App;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestSingleImportPerStatement(t *testing.T) {
	got, changed := apply(t, SingleImportPerStatement{}, "<?php\nuse A\\B, C\\D;")
	if want := "<?php\nuse A\\B;\nuse D;"; got == want {
		t.Fatalf("unexpected: %q", got) // guard against accidental truncation
	}
	if want := "<?php\nuse A\\B;\nuse C\\D;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// closure use and group import are left alone
	if _, changed := apply(t, SingleImportPerStatement{}, "<?php $f = function () use ($a, $b) {};"); changed {
		t.Fatal("closure use must not be split")
	}
	if _, changed := apply(t, SingleImportPerStatement{}, "<?php use A\\{B, C};"); changed {
		t.Fatal("group import must not be split")
	}
}

func TestNoBlankLinesAfterClassOpening(t *testing.T) {
	got, changed := apply(t, NoBlankLinesAfterClassOpening{}, "<?php class A\n{\n\n\n    public $x;\n}")
	if want := "<?php class A\n{\n    public $x;\n}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestIndentationType(t *testing.T) {
	got, changed := apply(t, IndentationType{}, "<?php\n\t$a = 1;\n\t\t$b = 2;\n")
	if want := "<?php\n    $a = 1;\n        $b = 2;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestFullOpeningTag(t *testing.T) {
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
}

func TestNoClosingTag(t *testing.T) {
	got, changed := apply(t, NoClosingTag{}, "<?php echo 1;\n?>\n")
	if want := "<?php echo 1;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// templating file (inline HTML) keeps the closing tag
	if _, changed := apply(t, NoClosingTag{}, "<?php if ($a): ?>\n<div></div>\n<?php endif; ?>\n"); changed {
		t.Fatal("file with inline HTML must keep closing tags")
	}
}

func TestBlankLineAfterNamespace(t *testing.T) {
	got, changed := apply(t, BlankLineAfterNamespace{}, "<?php\nnamespace App;\nclass A {}")
	if want := "<?php\nnamespace App;\n\nclass A {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, BlankLineAfterNamespace{}, "<?php\nnamespace App { }"); changed {
		t.Fatal("bracketed namespace must be left alone")
	}
}

func TestSingleLineAfterImports(t *testing.T) {
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
	got, changed := apply(t, NoTrailingWhitespace{}, "<?php $x = 1;   \n$y = 2;\t\n")
	if want := "<?php $x = 1;\n$y = 2;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, NoTrailingWhitespace{}, "<?php $x = 1;\n"); changed {
		t.Fatal("clean input should not change")
	}
}

func TestSingleBlankLineAtEndOfFile(t *testing.T) {
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
