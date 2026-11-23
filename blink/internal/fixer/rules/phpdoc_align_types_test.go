package rules

import "testing"

// A tag line is aligned only when its type matches PHP-CS-Fixer's TypeExpression::REGEX_TYPES.
// The expected output of every case comes from the PHP engine.
func TestPhpdocAlignTypeHints(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, src, want string }{
		{
			"a generic with only a wildcard is no type: the line stays out of the alignment",
			`<?php

final readonly class Notifier
{
    /**
     * @param class-string<ProviderInterface<*>> $providerClass
     * @param string|null $label      The label
     * @param bool        $sendEmails Whether emails are sent
     */
    public function __construct(
        public string $providerClass,
        public ?string $label = null,
        public bool $sendEmails = true,
    ) {}
}
`,
			`<?php

final readonly class Notifier
{
    /**
     * @param class-string<ProviderInterface<*>> $providerClass
     * @param string|null $label      The label
     * @param bool        $sendEmails Whether emails are sent
     */
    public function __construct(
        public string $providerClass,
        public ?string $label = null,
        public bool $sendEmails = true,
    ) {}
}
`,
		},
		{
			"the line splits the group: the tags before and after it are aligned apart",
			`<?php
/**
 * @param int $a
 * @param array<*> $b
 * @param string $cc
 * @param bool $d
 */
function f($a, $b, $cc, $d) {}
`,
			`<?php
/**
 * @param int $a
 * @param array<*> $b
 * @param string $cc
 * @param bool   $d
 */
function f($a, $b, $cc, $d) {}
`,
		},
		{
			"a wildcard after a comma is read as a docblock star: the type matches",
			`<?php
/**
 * @param array<int, *> $a
 * @param string $bb
 */
function f($a, $bb) {}
`,
			`<?php
/**
 * @param array<int, *> $a
 * @param string        $bb
 */
function f($a, $bb) {}
`,
		},
		{
			"a return type that is no type, after a single space: the line is no tag",
			`<?php
/**
 * @param string $a
 * @return key-of<*>
 */
function f($a) {}
`,
			`<?php
/**
 * @param string $a
 * @return key-of<*>
 */
function f($a) {}
`,
		},
		{
			"a return type that is no type, after two spaces: an empty hint and the rest as description",
			`<?php
/**
 * @param string $a
 * @return  Foo<*> the result
 */
function f($a) {}
`,
			`<?php
/**
 * @param  string $a
 * @return        Foo<*> the result
 */
function f($a) {}
`,
		},
		{
			"a callable return type keeps its own return type",
			`<?php
/**
 * @param int $a
 * @return Closure(): void the callback
 */
function f($a) {}
`,
			`<?php
/**
 * @param  int             $a
 * @return Closure(): void the callback
 */
function f($a) {}
`,
		},
		{
			"a union with spaces around its glue is one type",
			`<?php
/**
 * @param int $a the first
 * @return T | null the value
 *   or null
 */
function f($a) {}
`,
			`<?php
/**
 * @param  int      $a the first
 * @return T | null the value
 *                  or null
 */
function f($a) {}
`,
		},
		{
			"a generic that a space does not follow gives way to its name",
			`<?php
/**
 * @param int $a
 * @return mixed <b>TRUE</b> on success
 */
function f($a) {}
`,
			`<?php
/**
 * @param  int   $a
 * @return mixed <b>TRUE</b> on success
 */
function f($a) {}
`,
		},
		{
			"angle brackets in a quoted literal",
			`<?php
/**
 * @param '>'|'<' $op
 * @param int $a
 */
function f($op, $a) {}
`,
			`<?php
/**
 * @param '>'|'<' $op
 * @param int     $a
 */
function f($op, $a) {}
`,
		},
		{
			"the variable may follow the type without a space",
			`<?php
/**
 * @param Foo$x
 * @param int $long
 */
function f($x, $long) {}
`,
			`<?php
/**
 * @param Foo $x
 * @param int $long
 */
function f($x, $long) {}
`,
		},
		{
			"$this is a type",
			`<?php
/**
 * @param $this $x
 * @param int $long
 */
function f($x, $long) {}
`,
			`<?php
/**
 * @param $this $x
 * @param int   $long
 */
function f($x, $long) {}
`,
		},
		{
			"a callable needs a type before its variadic dots",
			`<?php
/**
 * @param int $a
 * @param callable(...): int $cb
 * @param string $bb
 */
function f($a, $cb, $bb) {}
`,
			`<?php
/**
 * @param int $a
 * @param callable(...): int $cb
 * @param string $bb
 */
function f($a, $cb, $bb) {}
`,
		},
		{
			"a list of classes, an array key type and a variable are no types",
			`<?php
class A
{
    /**
     * @param int $a
     * @throws FooException, BarException
     * @return $result the value
     * @var iterable[callable]
     */
    public $a;
}
`,
			`<?php
class A
{
    /**
     * @param int $a
     * @throws FooException, BarException
     * @return $result the value
     * @var iterable[callable]
     */
    public $a;
}
`,
		},
		{
			"types with spaces inside brackets",
			`<?php
/**
 * @param int<0, max> $a
 * @param array{a: int, b?: string} $b
 * @return ($a is 0 ? null : string) the name
 */
function f($a, $b) {}
`,
			`<?php
/**
 * @param  int<0, max>               $a
 * @param  array{a: int, b?: string} $b
 * @return ($a is 0 ? null : string) the name
 */
function f($a, $b) {}
`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assertFix(t, PhpdocAlign{}, c.src, c.want, c.src != c.want)
		})
	}
}
