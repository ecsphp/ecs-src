package rules

import "testing"

// Cases ported from PHP-CS-Fixer's BracesPositionFixerTest (tests/Fixer/Basic);
// every expected output is what PHP-CS-Fixer produces for the input.
func TestBracesPositionPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, BracesPosition{}, []aliasUpstreamCase{
		{"if (default)", nil,
			`<?php
                if ($foo)
                {
                    foo();
                }`,
			`<?php
                if ($foo) {
                    foo();
                }`},
		{"if (next line)", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                if ($foo) {
                    foo();
                }`,
			`<?php
                if ($foo)
                {
                    foo();
                }`},
		{"else (default)", nil,
			`<?php
                if ($foo)
                {
                    foo();
                }
                else
                {
                    bar();
                }`,
			`<?php
                if ($foo) {
                    foo();
                }
                else {
                    bar();
                }`},
		{"else (next line)", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                if ($foo) {
                    foo();
                }
                else {
                    bar();
                }`,
			`<?php
                if ($foo)
                {
                    foo();
                }
                else
                {
                    bar();
                }`},
		{"elseif (default)", nil,
			`<?php
                if ($foo)
                {
                    foo();
                }
                elseif ($bar)
                {
                    bar();
                }`,
			`<?php
                if ($foo) {
                    foo();
                }
                elseif ($bar) {
                    bar();
                }`},
		{"elseif (next line)", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                if ($foo) {
                    foo();
                }
                elseif ($bar) {
                    bar();
                }`,
			`<?php
                if ($foo)
                {
                    foo();
                }
                elseif ($bar)
                {
                    bar();
                }`},
		{"else if (default)", nil,
			`<?php
                if ($foo)
                {
                    foo();
                }
                else if ($bar)
                {
                    bar();
                }`,
			`<?php
                if ($foo) {
                    foo();
                }
                else if ($bar) {
                    bar();
                }`},
		{"else if (next line)", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                if ($foo) {
                    foo();
                }
                else if ($bar) {
                    bar();
                }`,
			`<?php
                if ($foo)
                {
                    foo();
                }
                else if ($bar)
                {
                    bar();
                }`},
		{"for (default)", nil,
			`<?php
                for (;;)
                {
                    foo();
                }`,
			`<?php
                for (;;) {
                    foo();
                }`},
		{"for (next line)", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                for (;;) {
                    foo();
                }`,
			`<?php
                for (;;)
                {
                    foo();
                }`},
		{"foreach (default)", nil,
			`<?php
                foreach ($foo as $bar)
                {
                    foo();
                }`,
			`<?php
                foreach ($foo as $bar) {
                    foo();
                }`},
		{"foreach (next line)", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                foreach ($foo as $bar) {
                    foo();
                }`,
			`<?php
                foreach ($foo as $bar)
                {
                    foo();
                }`},
		{"while (default)", nil,
			`<?php
                while ($foo)
                {
                    foo();
                }`,
			`<?php
                while ($foo) {
                    foo();
                }`},
		{"while (next line)", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                while ($foo) {
                    foo();
                }`,
			`<?php
                while ($foo)
                {
                    foo();
                }`},
		{"do while (default)", nil,
			`<?php
                do
                {
                    foo();
                } while ($foo);`,
			`<?php
                do {
                    foo();
                } while ($foo);`},
		{"do while (next line)", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                do {
                    foo();
                } while ($foo);`,
			`<?php
                do
                {
                    foo();
                } while ($foo);`},
		{"switch (default)", nil,
			`<?php
                switch ($foo)
                {
                    case 1:
                        foo();
                }`,
			`<?php
                switch ($foo) {
                    case 1:
                        foo();
                }`},
		{"switch (next line)", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                switch ($foo) {
                    case 1:
                        foo();
                }`,
			`<?php
                switch ($foo)
                {
                    case 1:
                        foo();
                }`},
		{"try catch finally (open brace on the same line)", nil,
			`<?php
	                try
	                {
	                    foo();
	                }
	                catch (Exception $e)
	                {
	                    bar();
	                }
	                finally
	                {
	                    baz();
	                }`,
			`<?php
	                try {
	                    foo();
	                }
	                catch (Exception $e) {
	                    bar();
	                }
	                finally {
	                    baz();
	                }`},
		{"try catch finally (open brace on the next line)", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
	                try {
	                    foo();
	                }
	                catch (Exception $e) {
	                    bar();
	                }
	                finally {
	                    baz();
	                }`,
			`<?php
	                try
	                {
	                    foo();
	                }
	                catch (Exception $e)
	                {
	                    bar();
	                }
	                finally
	                {
	                    baz();
	                }`},
		{"class (default)", nil,
			`<?php
                class Foo {
                }`,
			`<?php
                class Foo
                {
                }`},
		{"class (same line)", map[string]any{"classes_opening_brace": "same_line"},
			`<?php
                class Foo
                {
                }`,
			`<?php
                class Foo {
                }`},
		{"function (default)", nil,
			`<?php
                function foo() {
                }`,
			`<?php
                function foo()
                {
                }`},
		{"function (same line)", map[string]any{"functions_opening_brace": "same_line"},
			`<?php
                function foo()
                {
                }`,
			`<?php
                function foo() {
                }`},
		{"anonymous function (default)", nil,
			`<?php
                $foo = function ()
                {
                };`,
			`<?php
                $foo = function () {
                };`},
		{"anonymous function (next line)", map[string]any{"anonymous_functions_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                $foo = function () {
                };`,
			`<?php
                $foo = function ()
                {
                };`},
		{"with blank lines inside braces", nil,
			`<?php
                class Foo {

                    public function foo() {

                        if (true)
                        {

                            echo "foo";

                        }

                    }

                }`,
			`<?php
                class Foo
                {

                    public function foo()
                    {

                        if (true) {

                            echo "foo";

                        }

                    }

                }`},
		{"with comment after opening brace (default)", nil,
			`<?php
                function foo() /* foo */ { // foo
                }`,
			`<?php
                function foo() /* foo */ // foo
                {
                }`},
		{"with comment after opening brace (same line)", map[string]any{"functions_opening_brace": "same_line"},
			`<?php
                function foo() // foo
                { /* foo */
                }`,
			`<?php
                function foo() { // foo
                /* foo */
                }`},
		{"next line with multiline signature", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                if (
                    $foo
                )
                {
                    foo();
                }`,
			`<?php
                if (
                    $foo
                ) {
                    foo();
                }`},
		{"next line with newline before closing parenthesis", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                if ($foo
                )
                {
                    foo();
                }`,
			`<?php
                if ($foo
                ) {
                    foo();
                }`},
		{"next line with newline in signature but not before closing parenthesis", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                if (
                    $foo) {
                    foo();
                }`,
			`<?php
                if (
                    $foo)
                {
                    foo();
                }`},
		{"anonymous class (same line)", nil,
			`<?php
                $foo = new class()
                {
                };`,
			`<?php
                $foo = new class() {
                };`},
		{"anonymous class (next line)", map[string]any{"anonymous_classes_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                $foo = new class() {
                };`,
			`<?php
                $foo = new class()
                {
                };`},
		{"next line with multiline signature and return type", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                function foo(
                    $foo
                ): int
                {
                    foo();
                }`,
			`<?php
                function foo(
                    $foo
                ): int {
                    foo();
                }`},
		{"next line with multiline signature and return type (nullable)", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                function foo(
                    $foo
                ): ?int
                {
                    foo();
                }`,
			`<?php
                function foo(
                    $foo
                ): ?int {
                    foo();
                }`},
		{"next line with multiline signature and return type (array)", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                function foo(
                    $foo
                ): array
                {
                    foo();
                }`,
			`<?php
                function foo(
                    $foo
                ): array {
                    foo();
                }`},
		{"next line with multiline signature and return type (class name)", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                function foo(
                    $foo
                ): \Foo\Bar
                {
                    foo();
                }`,
			`<?php
                function foo(
                    $foo
                ): \Foo\Bar {
                    foo();
                }`},
		{"next line with newline before closing parenthesis and return type", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                function foo($foo
                ): int
                {
                    foo();
                }`,
			`<?php
                function foo($foo
                ): int {
                    foo();
                }`},
		{"next line with newline before closing parenthesis and callable type", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                function foo($foo
                ): callable
                {
                    return function (): void {};
                }`,
			`<?php
                function foo($foo
                ): callable {
                    return function (): void {};
                }`},
		{"next line with newline in signature but not before closing parenthesis and return type", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
                function foo(
                    $foo): int {
                    foo();
                }`,
			`<?php
                function foo(
                    $foo): int
                {
                    foo();
                }`},
		{"multiple elseifs", nil,
			`<?php if ($foo){
                } elseif ($foo){
                } elseif ($foo){
                } elseif ($foo){
                } elseif ($foo){
                } elseif ($foo){
                } elseif ($foo){
                } elseif ($foo){
                } elseif ($foo){
                }`,
			`<?php if ($foo) {
                } elseif ($foo) {
                } elseif ($foo) {
                } elseif ($foo) {
                } elseif ($foo) {
                } elseif ($foo) {
                } elseif ($foo) {
                } elseif ($foo) {
                } elseif ($foo) {
                }`},
		{"open brace preceded by comment and whitespace", nil,
			`<?php
                if (true) /* foo */ {
                    foo();
                }`,
			`<?php
                if (true) { /* foo */
                    foo();
                }`},
		{"open brace surrounded by comment and whitespace", nil,
			`<?php
                if (true) /* foo */ { /* bar */
                    foo();
                }`,
			`<?php
                if (true) { /* foo */ /* bar */
                    foo();
                }`},
		{"open brace not preceded by space and followed by a comment", nil,
			`<?php class test
{
    public function example(){// example
    }
}
`,
			`<?php class test
{
    public function example()// example
    {
    }
}
`},
		{"open brace not preceded by space and followed by a space and comment", nil,
			`<?php class test
{
    public function example(){ // example
    }
}
`,
			`<?php class test
{
    public function example() // example
    {
    }
}
`},
		{"provideFixCases #0", nil,
			`<?php
if ($x)//
{

    f();
}`,
			`<?php
if ($x) {//

    f();
}`},
		{"variable terminated by close tag", nil,
			`<?php

$pager = new class { public function bar() { return 'baz'; } };

echo $pager->bar()
?>`,
			`<?php

$pager = new class {
public function bar()
{
return 'baz';
}
};

echo $pager->bar()
?>`},
		{"confirm {$foo} does not cause harm", nil,
			`<?php
$foo = "foo";
if($foo) {
    echo "
/* some sql */ {$foo} /* some sql */;
    ";
}`,
			`<?php
$foo = "foo";
if($foo) {
    echo "
/* some sql */ {$foo} /* some sql */;
    ";
}`},
		{"newline after opening brace", nil,
			`<?php if (true) {}`,
			`<?php if (true) {
}`},
		{"newline before closing comment and parenthesis", map[string]any{"functions_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
function foo($foo
/* comment */) {
}`,
			`<?php
function foo($foo
/* comment */) {
}`},
		{"anonymous function", map[string]any{"anonymous_functions_opening_brace": "same_line", "allow_single_line_anonymous_functions": true},
			`<?php $f = function(){};`,
			`<?php $f = function() {};`},
		{"comment between closing parenthesis and opening brace", map[string]any{"control_structures_opening_brace": "same_line"},
			`<?php if ($a) /* comment */ {}`,
			`<?php if ($a) { /* comment */
}`},
		{"function (multiline + union return)", nil,
			`<?php
                function sum(
                    int|float $first,
                    int|float $second,
                ): int|float
                {
                }`,
			`<?php
                function sum(
                    int|float $first,
                    int|float $second,
                ): int|float {
                }`},
		{"function (multiline + union return with whitespace)", nil,
			`<?php
                function sum(
                    int|float $first,
                    int|float $second,
                ): int | float
                {
                }`,
			`<?php
                function sum(
                    int|float $first,
                    int|float $second,
                ): int | float {
                }`},
		{"method with static return type", nil,
			`<?php
                class Foo
                {
                    function sum(
                        $foo
                    ): static
                    {
                    }
                }`,
			`<?php
                class Foo
                {
                    function sum(
                        $foo
                    ): static {
                    }
                }`},
		{"function (multiline + intersection return)", nil,
			`<?php
                function foo(
                    mixed $bar,
                    mixed $baz,
                ): Foo&Bar {
                }`,
			`<?php
                function foo(
                    mixed $bar,
                    mixed $baz,
                ): Foo&Bar {
                }`},
		{"function (multiline + DNF return)", nil,
			`<?php
                function foo(
                    mixed $bar,
                    mixed $baz,
                ): (Foo&Bar)|int|null {
                }`,
			`<?php
                function foo(
                    mixed $bar,
                    mixed $baz,
                ): (Foo&Bar)|int|null {
                }`},
		{"property hook one-liners", nil,
			`<?php abstract class Foo
{
    abstract public bool $b { get; set; }

    abstract public int $i { set(int $i) { $this->i = $i + 10; } get; }
}`,
			`<?php abstract class Foo
{
    abstract public bool $b { get; set; }

    abstract public int $i { set(int $i) { $this->i = $i + 10; } get; }
}`},
		{"property hook", nil,
			`<?php class C
{
    private int $i    {
        get => 2;
    }
}`,
			`<?php class C
{
    private int $i {
        get => 2;
    }
}`},
		{"property hook with default", nil,
			`<?php class C
{
    private mixed $wat = [-7, "a" . 'b', null, array(), 1/3, self::FOO, ]        {
        set(mixed $wat) {
            $this->wat = $wat;
        }
    }

    private mixed $wat2 = array(-7, "a" . 'b', null, array(), 1/3, self::FOO, )        {
        set(mixed $wat2) {
            $this->wat2 = $wat2;
        }
    }
}`,
			`<?php class C
{
    private mixed $wat = [-7, "a" . 'b', null, array(), 1/3, self::FOO, ] {
        set(mixed $wat) {
            $this->wat = $wat;
        }
    }

    private mixed $wat2 = array(-7, "a" . 'b', null, array(), 1/3, self::FOO, ) {
        set(mixed $wat2) {
            $this->wat2 = $wat2;
        }
    }
}`},
		{"property hook in promoted property", nil,
			`<?php class CarPark
{
    public function __construct(
        public Car $car {
            set(Car $car)
            {
                $this->car = $car;
                $this->car->parked();
            }
        },
    ) {}
}`,
			`<?php class CarPark
{
    public function __construct(
        public Car $car {
            set(Car $car) {
                $this->car = $car;
                $this->car->parked();
            }
        },
    ) {}
}`},
		{"property hook in promoted property with default", nil,
			`<?php class IntVal
{
    public function __construct(
        public int $int = 5 {
            set(int $int)
                                    {
                $this->int = $int;
            }
        },
    ) {}
}`,
			`<?php class IntVal
{
    public function __construct(
        public int $int = 5 {
            set(int $int) {
                $this->int = $int;
            }
        },
    ) {}
}`},
	})
}

// A body written on the line of its braces, on each construct the fixer handles;
// every expected output is what PHP-CS-Fixer produces for the input.
func TestBracesPositionBodyOnOwnLines(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, BracesPosition{}, []aliasUpstreamCase{
		{"issue example", nil,
			`<?php

if ($enabled) { foo($input); }

function bar(): int { return 1; }

if ($enabled) {
    foo($input); }
`,
			`<?php

if ($enabled) {
foo($input);
}

function bar(): int
{
return 1;
}

if ($enabled) {
    foo($input);
}
`},
		{"empty bodies", nil,
			`<?php
class A {}
interface I {}
enum E {}
function f() {}
if ($a) {}
while ($a) {}
$f = function () {};
$c = new class {};
`,
			`<?php
class A
{
}
interface I
{
}
enum E
{
}
function f()
{
}
if ($a) {
}
while ($a) {
}
$f = function () {};
$c = new class {};
`},
		{"control structures", nil,
			`<?php
foreach ($a as $b) { c(); }
try { a(); } catch (E $e) { b(); } finally { c(); }
do { a(); } while ($x);
switch ($a) { case 1: break; }
$m = match ($a) { 1 => 2, default => 3 };
declare(ticks=1) { a(); }
if ($a) { b(); } elseif ($c) { d(); } else { e(); }
`,
			`<?php
foreach ($a as $b) {
c();
}
try {
a();
} catch (E $e) {
b();
} finally {
c();
}
do {
a();
} while ($x);
switch ($a) {
case 1: break;
}
$m = match ($a) {
1 => 2, default => 3
};
declare(ticks=1) {
a();
}
if ($a) {
b();
} elseif ($c) {
d();
} else {
e();
}
`},
		{"class and method on one line", nil,
			`<?php
class A { public function b() { return 1; } }
    trait T { public $p; }
`,
			`<?php
class A
{
public function b()
{
return 1;
}
}
    trait T
    {
    public $p;
    }
`},
		{"anonymous class with a body", nil,
			`<?php
$d = new class { public $x; };
$e = new class($a) extends B { };
`,
			`<?php
$d = new class {
public $x;
};
$e = new class($a) extends B { };
`},
		{"anonymous class, no single line", map[string]any{"allow_single_line_empty_anonymous_classes": false},
			`<?php
$c = new class {};
$d = new class { public $x; };
`,
			`<?php
$c = new class {
};
$d = new class {
public $x;
};
`},
		{"closure in a control structure", nil,
			`<?php
if ($a) { $k = function () { if ($b) { c(); } }; }
`,
			`<?php
if ($a) {
$k = function () { if ($b) { c(); } };
}
`},
		{"single-line closures off", map[string]any{"allow_single_line_anonymous_functions": false},
			`<?php
$g = function () { if ($a) { b(); } };
`,
			`<?php
$g = function () {
if ($a) {
b();
}
};
`},
		{"next line everywhere", map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end", "anonymous_functions_opening_brace": "next_line_unless_newline_at_signature_end", "anonymous_classes_opening_brace": "next_line_unless_newline_at_signature_end"},
			`<?php
if ($a) { b(); }
$f = function () { return 1; };
$o = new class { public $p; };
`,
			`<?php
if ($a)
{
b();
}
$f = function ()
{ return 1; };
$o = new class
{
public $p;
};
`},
		{"same line for classes and functions", map[string]any{"classes_opening_brace": "same_line", "functions_opening_brace": "same_line"},
			`<?php
class A { public function b() { return 1; } }
`,
			`<?php
class A {
public function b() {
return 1;
}
}
`},
		{"comments on the brace line", nil,
			`<?php
if ($a) { // c
    b(); }
if ($a) { /* c */ b(); }
function f() { /** d */ return 1; }
`,
			`<?php
if ($a) { // c
    b();
}
if ($a) { /* c */ b();
}
function f()
{ /** d */ return 1;
}
`},
		{"close tag", nil,
			`<?php
if ($a) { ?>x<?php }
if ($a) { b(); ?>x<?php }
`,
			`<?php
if ($a) { ?>x<?php }
if ($a) {
b(); ?>x<?php }
`},
		{"closing brace after code", nil,
			`<?php
class A
{
    public function b()
    {
        if ($a) {
            return 1; }
        return 2; }
}
`,
			`<?php
class A
{
    public function b()
    {
        if ($a) {
            return 1;
        }
        return 2;
    }
}
`},
		{"property hooks", nil,
			`<?php
class A
{
    public int $a { get => 1; }
    public int $b { get { return 1; } }
    public int $c {
        get { return 1; } set (int $v) { $this->c = $v; } }
}
`,
			`<?php
class A
{
    public int $a { get => 1; }
    public int $b { get { return 1; } }
    public int $c {
        get { return 1; } set (int $v) { $this->c = $v; }
    }
}
`},
		{"promoted property with hooks", nil,
			`<?php
class A
{
    public function __construct(
        public int $a { set (int $v) { $this->a = $v; } },
    ) { foo(); }
}
`,
			`<?php
class A
{
    public function __construct(
        public int $a { set (int $v) { $this->a = $v; } },
    ) { foo(); }
}
`},
	})
}

// PHP-CS-Fixer changes these again on a second run: a body is left on one line
// only inside the first body left so in the file, and a property hook list is
// broken into lines once a hook body in it spans lines. Both runs are checked
// against what PHP-CS-Fixer produces.
func TestBracesPositionSecondRun(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		cfg    map[string]any
		input  string
		first  string
		second string
	}{
		{"inside the first single-line closure", nil,
			`<?php
$g = function () { if ($a) { b(); } };
$h = function () { if ($a) { b(); } };
`,
			`<?php
$g = function () { if ($a) { b(); } };
$h = function () { if ($a) {
b();
} };
`,
			`<?php
$g = function () { if ($a) { b(); } };
$h = function () {
if ($a) {
b();
}
};
`},
		{"empty closure first", nil,
			`<?php
$f = function () {};
$g = function () { if ($a) { b(); } };
`,
			`<?php
$f = function () {};
$g = function () { if ($a) {
b();
} };
`,
			`<?php
$f = function () {};
$g = function () {
if ($a) {
b();
}
};
`},
		{"property hooks, no single line", map[string]any{"allow_single_line_anonymous_functions": false},
			`<?php
class A
{
    public int $b { get { return 1; } }
}
`,
			`<?php
class A
{
    public int $b { get {
    return 1;
    } }
}
`,
			`<?php
class A
{
    public int $b {
    get {
    return 1;
    }
    }
}
`},
	}
	for _, c := range cases {
		f := BracesPosition{}.WithConfig(c.cfg).(fixerRule)
		if got, _ := apply(t, f, c.input); got != c.first {
			t.Errorf("%s, first run:\n got: %q\nwant: %q", c.name, got, c.first)
		}
		if got, _ := apply(t, f, c.first); got != c.second {
			t.Errorf("%s, second run:\n got: %q\nwant: %q", c.name, got, c.second)
		}
	}
}
