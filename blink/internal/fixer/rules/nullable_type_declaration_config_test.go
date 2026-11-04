package rules

import (
	"slices"
	"strings"
	"testing"

	"blink/internal/lexer"
	"blink/internal/token"
	"blink/internal/tokens"
)

func TestNullableTypeDeclarationConfig(t *testing.T) {
	t.Parallel()

	src := "<?php function f(?int $x): ?string {}"

	// default (question_mark) leaves the "?" syntax as is.
	if _, changed := apply(t, NullableTypeDeclaration{}, src); changed {
		t.Fatal("question_mark default must leave ?int / ?string untouched")
	}

	// union rewrites "?T" to "null|T", null first as PHP-CS-Fixer writes it.
	f := NullableTypeDeclaration{}.WithConfig(map[string]any{"syntax": "union"})
	got, changed := apply(t, f.(fixerRule), src)
	if want := "<?php function f(null|int $x): null|string {}"; !changed || got != want {
		t.Fatalf("union changed=%v got=%q want=%q", changed, got, want)
	}

	// union still collapses nothing when the type is already a union.
	already := "<?php function f(int|null $x) {}"
	if _, changed := apply(t, f.(fixerRule), already); changed {
		t.Fatal("union must leave an existing union alone")
	}
}

// Expected outputs come from the PHP engine.
func TestNullableTypeDeclarationElements(t *testing.T) {
	t.Parallel()
	union := NullableTypeDeclaration{syntax: "union"}
	question := NullableTypeDeclaration{}
	cases := []struct {
		name string
		f    NullableTypeDeclaration
		in   string
		want string
	}{
		{
			"union: properties, promoted and attributed parameters",
			union,
			"<?php class A {\n    public private(set) ?int $a = null;\n    protected(set) ?string $b;\n    public function __construct(public private(set) ?int $c = null, #[X] ?int $d = null) {}\n    public ?\\Foo\\Bar $g;\n    public static ?self $h;\n    var ?array $i;\n}",
			"<?php class A {\n    public private(set) null|int $a = null;\n    protected(set) null|string $b;\n    public function __construct(public private(set) null|int $c = null, #[X] null|int $d = null) {}\n    public null|\\Foo\\Bar $g;\n    public static null|self $h;\n    var null|array $i;\n}",
		},
		{
			"union: typed constants are not elements",
			union,
			"<?php class A { const ?int X = 1; final public const ?int Y = 1; }",
			"<?php class A { const ?int X = 1; final public const ?int Y = 1; }",
		},
		{
			"union: property hook parameter is not an element",
			union,
			"<?php class A { public $j { set(?int $v) { $this->j = $v; } } }",
			"<?php class A { public $j { set(?int $v) { $this->j = $v; } } }",
		},
		{
			"union: space put before the type",
			union,
			"<?php function g(int $a,?int $b):?int {}\nfunction h(#[X]?int $x, ? int $y, ?int &$z, ?int ...$w): ? static {}\nfunction k(?/*c*/int $a, /*d*/?int $b) {}",
			"<?php function g(int $a, null|int $b): null|int {}\nfunction h(#[X] null|int $x, null|int $y, null|int &$z, null|int ...$w): null|static {}\nfunction k(null|int $a, /*d*/ null|int $b) {}",
		},
		{
			"union: closures, arrow functions and interfaces",
			union,
			"<?php $c = function (?int $a) use ($b): ?int {};\n$d = fn (?int $a): ?int => $a;\ninterface I { public function m(?int $a): ?int; }\n$e = $a ? $b : $c;",
			"<?php $c = function (null|int $a) use ($b): null|int {};\n$d = fn (null|int $a): null|int => $a;\ninterface I { public function m(null|int $a): null|int; }\n$e = $a ? $b : $c;",
		},
		{
			"union: reserved-word method name",
			union,
			"<?php class A { function &list(?int $a): ?int {} }",
			"<?php class A { function &list(null|int $a): null|int {} }",
		},
		{
			"question mark: properties, promoted and attributed parameters",
			question,
			"<?php class A {\n    public private(set) int|null $a = null;\n    protected(set) string|null $b;\n    public function __construct(public private(set) int|null $c = null, #[X] int|null $d = null) {}\n    public static self|null $h;\n    public int|null $e, $f;\n}",
			"<?php class A {\n    public private(set) ?int $a = null;\n    protected(set) ?string $b;\n    public function __construct(public private(set) ?int $c = null, #[X] ?int $d = null) {}\n    public static ?self $h;\n    public ?int $e, $f;\n}",
		},
		{
			"question mark: typed constants and hook parameters are not elements",
			question,
			"<?php class A { const int|null X = null; public $j { set(int|null $v) {} } }",
			"<?php class A { const int|null X = null; public $j { set(int|null $v) {} } }",
		},
		{
			"question mark: space put before the type",
			question,
			"<?php function g(int $a,int|null $b):int|null {}\nfunction h(#[X]int|null $x, int | null $y, NULL|int &...$z) {}\nfunction k(/*d*/int|null $b) {}",
			"<?php function g(int $a, ?int $b): ?int {}\nfunction h(#[X] ?int $x, ?int $y, ?int &...$z) {}\nfunction k(/*d*/ ?int $b) {}",
		},
		{
			"question mark: multiline type",
			question,
			"<?php function g(int|\n    null $a): null\n    |int {}",
			"<?php function g(?int $a): ?int {}",
		},
		{
			"question mark: DNF, false and three-member unions",
			question,
			"<?php function f((A&B)|null $a, false|null $b, int|string|null $c, A&B $d) {}",
			"<?php function f((A&B)|null $a, ?false $b, int|string|null $c, A&B $d) {}",
		},
		{
			"question mark: use function import and catch",
			question,
			"<?php use function Foo\\bar;\ntry {} catch (A|B $e) {}\nbar(null|$x);",
			"<?php use function Foo\\bar;\ntry {} catch (A|B $e) {}\nbar(null|$x);",
		},
	}
	for _, c := range cases {
		got, _ := apply(t, c.f, c.in)
		if got != c.want {
			t.Errorf("%s\n  in: %q\n got: %q\nwant: %q", c.name, c.in, got, c.want)
		}
	}
}

// TestAsymmetricVisibilitySingleToken checks the classifiers on the single
// keyword token the lexer emits for "private(set)" (PHP 8.4's T_PRIVATE_SET).
func TestAsymmetricVisibilitySingleToken(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		f    fixerRule
		in   string
		want string
	}{
		{
			"NullableTypeDeclaration union",
			NullableTypeDeclaration{syntax: "union"},
			"<?php class A { public private(set) ?int $a; private(set) ?int $b; public function __construct(protected(set) ?int $c) {} }",
			"<?php class A { public private(set) null|int $a; private(set) null|int $b; public function __construct(protected(set) null|int $c) {} }",
		},
		{
			"NullableTypeDeclaration question mark",
			NullableTypeDeclaration{},
			"<?php class A { public private(set) int|null $a; private(set) int|null $b; public function __construct(protected(set) int|null $c) {} }",
			"<?php class A { public private(set) ?int $a; private(set) ?int $b; public function __construct(protected(set) ?int $c) {} }",
		},
		{
			"TernaryOperatorSpaces",
			TernaryOperatorSpaces{},
			"<?php class A { public private(set) ?int $a; public function __construct(private(set) ?int $c) {} }",
			"<?php class A { public private(set) ?int $a; public function __construct(private(set) ?int $c) {} }",
		},
		{
			"OperatorLinebreak",
			OperatorLinebreak{},
			"<?php\nclass A { public function __construct(public private(set) ?\n    int $a) {} }\n",
			"<?php\nclass A { public function __construct(public private(set) ?\n    int $a) {} }\n",
		},
	}
	for _, c := range cases {
		toks := lexer.Lex(c.in)
		if !slices.ContainsFunc(toks, func(t token.Token) bool { return t.Kind == token.Keyword && strings.HasSuffix(t.Value, "(set)") }) {
			t.Fatalf("%s: no merged (set) token", c.name)
		}
		s := tokens.New(toks)
		c.f.Fix(s)
		if got := s.Render(); got != c.want {
			t.Errorf("%s\n  in: %q\n got: %q\nwant: %q", c.name, c.in, got, c.want)
		}
	}
}
