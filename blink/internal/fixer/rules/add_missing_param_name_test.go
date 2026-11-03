package rules

import "testing"

// TestAddMissingParamName adds the positional variable name to a type-only @param.
func TestAddMissingParamName(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass A {\n    /**\n     * @param string\n     */\n    function f($name) {}\n}\n"
	want := "<?php\nclass A {\n    /**\n     * @param string $name\n     */\n    function f($name) {}\n}\n"
	got, changed := apply(t, AddMissingParamName{}, src)
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, AddMissingParamName{}, got)
}

// TestAddMissingParamNameMultiLineArrayShape leaves a @param whose multi-line
// array-shape type already carries its name after the closing brace untouched.
func TestAddMissingParamNameMultiLineArrayShape(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass A {\n    /**\n     * @param array{\n     *     timed: list<array{category: array{label: string}, lines: list<string>}>,\n     *     timeless: list<array{category: array{label: string}, lines: list<string>}>\n     * } $messages\n     */\n    function f(array $messages) {}\n}\n"
	if got, changed := apply(t, AddMissingParamName{}, src); changed || got != src {
		t.Fatalf("multi-line array-shape param with name must be left alone: changed=%v got=%q", changed, got)
	}
}

// TestAddMissingParamNameMatchesPHP pins outputs taken from the PHP fixer
// itself (ECS with only AddMissingParamNameFixer), quirks included.
func TestAddMissingParamNameMatchesPHP(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, src, want string }{
		{
			"Symplify fixture wrong6",
			`<?php

/**
 * @param string
 * @param string
 */
function another($class, $context)
{
}

/**
 * @param string $one
 * @param string
 * @param string $three
 */
function anotherFunction(string $one, string $two, string $three)
{
}

?>
`,
			`<?php

/**
 * @param string $class
 * @param string $context
 */
function another($class, $context)
{
}

/**
 * @param string $one
 * @param string $two
 * @param string $three
 */
function anotherFunction(string $one, string $two, string $three)
{
}

?>
`,
		},
		{
			"multi-line shapes, glued names, generics, descriptions",
			`<?php

final class Shapes
{
    /**
     * @param array{
     *     width: int,
     *     height: int,
     * } $size
     * @param string
     */
    public function area(array $size, string $unit, int $scale): int
    {
        return 0;
    }

    /**
     * @param list<float|null>$ratios
     * @param string
     */
    public function ratios(array $ratios, string $label): void
    {
    }

    /**
     * @param list<int>
     * @param string label
     */
    public function generic(array $ids, string $label): void
    {
    }

    /**
     * @param string Some description
     * @param int
     */
    public function described(string $title, int $count): void
    {
    }
}
`,
			`<?php

final class Shapes
{
    /**
     * @param array{
     *     width: int,
     *     height: int,
     * } $size
     * @param string $unit
     */
    public function area(array $size, string $unit, int $scale): int
    {
        return 0;
    }

    /**
     * @param list<float|null>$ratios
     * @param string $ratios
     */
    public function ratios(array $ratios, string $label): void
    {
    }

    /**
     * @param list<int>
     * @param string $ids label
     */
    public function generic(array $ids, string $label): void
    {
    }

    /**
     * @param string $title Some description
     * @param int $count
     */
    public function described(string $title, int $count): void
    {
    }
}
`,
		},
		{
			"a file with \"callable $x)\" is skipped whole",
			`<?php

/**
 * @param string
 */
function run($task, callable $done)
{
}
`,
			`<?php

/**
 * @param string
 */
function run($task, callable $done)
{
}
`,
		},
		{
			"plain comments, later functions and duplicate names",
			`<?php

final class Later
{
    /** @var int */
    private $n;

    /**
     * Some text
     */
    private $m;

    /* @param string
     */
    public function go($first)
    {
    }

    /**
     * @param int
     * @param int
     */
    public function same($a, $a2, $a)
    {
    }
}
`,
			`<?php

final class Later
{
    /** @var int */
    private $n;

    /**
     * Some text
     */
    private $m;

    /* @param string $first
     */
    public function go($first)
    {
    }

    /**
     * @param int $a
     * @param int $a2
     */
    public function same($a, $a2, $a)
    {
    }
}
`,
		},
		{
			"use function is an import, not the related function",
			"<?php\n/**\n * @param string\n */\nuse function sprintf;\necho sprintf('%s', $x);\nfunction g($q) {}\n",
			"<?php\n/**\n * @param string $q\n */\nuse function sprintf;\necho sprintf('%s', $x);\nfunction g($q) {}\n",
		},
		{
			"function inside a group use is an import",
			"<?php\nnamespace N;\n/**\n * @param string\n */\nuse Foo\\{Cls, function bar, const BAZ};\nbar($x);\nfunction g($q) {}\n",
			"<?php\nnamespace N;\n/**\n * @param string $q\n */\nuse Foo\\{Cls, function bar, const BAZ};\nbar($x);\nfunction g($q) {}\n",
		},
		{
			"use function with a group is an import",
			"<?php\nnamespace N;\n/**\n * @param string\n */\nuse function Foo\\{bar, baz};\nbar($x);\nfunction g($q) {}\n",
			"<?php\nnamespace N;\n/**\n * @param string $q\n */\nuse function Foo\\{bar, baz};\nbar($x);\nfunction g($q) {}\n",
		},
	}
	for _, c := range cases {
		got, _ := apply(t, AddMissingParamName{}, c.src)
		if got != c.want {
			t.Errorf("%s:\n got: %q\nwant: %q", c.name, got, c.want)
			continue
		}
		idempotent(t, AddMissingParamName{}, got)
	}
}
