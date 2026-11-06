package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_doc_to_property_type).
func TestPhpdocToPropertyTypePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpdocToPropertyType{}, []aliasUpstreamCase{
		{
			"int and traversable", nil,
			"<?php\nclass Foo {\n    /** @var int */\n    private $foo;\n    /** @var \\Traversable */\n    private $bar;\n}\n",
			"<?php\nclass Foo {\n    /** @var int */\n    private int $foo;\n    /** @var \\Traversable */\n    private \\Traversable $bar;\n}\n",
		},
		{
			"union property", nil,
			"<?php\nclass Foo {\n    /** @var int|string */\n    private $foo;\n}\n",
			"<?php\nclass Foo {\n    /** @var int|string */\n    private int|string $foo;\n}\n",
		},
		{
			"union_types false", map[string]any{"union_types": false},
			"<?php\nclass Foo {\n    /** @var int|string */\n    private $foo;\n    /** @var \\Traversable */\n    private $bar;\n}\n",
			"<?php\nclass Foo {\n    /** @var int|string */\n    private $foo;\n    /** @var \\Traversable */\n    private \\Traversable $bar;\n}\n",
		},
		{
			"nullable property", nil,
			"<?php\nclass Foo {\n    /** @var int|null */\n    private $foo;\n}\n",
			"<?php\nclass Foo {\n    /** @var int|null */\n    private ?int $foo;\n}\n",
		},
		{
			"already typed noop", nil,
			"<?php\nclass Foo {\n    /** @var int */\n    private int $foo;\n}\n",
			"<?php\nclass Foo {\n    /** @var int */\n    private int $foo;\n}\n",
		},
		{
			"callable excluded", nil,
			"<?php\nclass Foo {\n    /** @var callable */\n    private $foo;\n}\n",
			"<?php\nclass Foo {\n    /** @var callable */\n    private $foo;\n}\n",
		},
		{
			"grouped no varname skipped", nil,
			"<?php\nclass Foo {\n    /** @var int */\n    private $foo, $bar;\n}\n",
			"<?php\nclass Foo {\n    /** @var int */\n    private $foo, $bar;\n}\n",
		},
		{
			"static property", nil,
			"<?php\nclass Foo {\n    /** @var int */\n    private static $foo;\n}\n",
			"<?php\nclass Foo {\n    /** @var int */\n    private static int $foo;\n}\n",
		},
		{
			"trait property", nil,
			"<?php\ntrait Foo {\n    /** @var int */\n    private $foo;\n}\n",
			"<?php\ntrait Foo {\n    /** @var int */\n    private int $foo;\n}\n",
		},
		{
			"skip method bodies", nil,
			"<?php\nclass Foo {\n    public function bar() {\n        /** @var int */\n        $x = 1;\n    }\n    /** @var int */\n    private $foo;\n}\n",
			"<?php\nclass Foo {\n    public function bar() {\n        /** @var int */\n        $x = 1;\n    }\n    /** @var int */\n    private int $foo;\n}\n",
		},
		{
			"named var annotation", nil,
			"<?php\nclass Foo {\n    /** @var int $foo */\n    private $foo;\n}\n",
			"<?php\nclass Foo {\n    /** @var int $foo */\n    private int $foo;\n}\n",
		},
		{
			"class const before", nil,
			"<?php\n$x = Foo::class;\nclass Bar {\n    /** @var int */\n    private $y;\n}\n",
			"<?php\n$x = Foo::class;\nclass Bar {\n    /** @var int */\n    private int $y;\n}\n",
		},
		{
			"class const after", nil,
			"<?php\nclass Bar {\n    /** @var int */\n    private $y;\n}\n$x = Foo::class;\n",
			"<?php\nclass Bar {\n    /** @var int */\n    private int $y;\n}\n$x = Foo::class;\n",
		},
		{
			"phpstan-type exclude", nil,
			"<?php\n/** @phpstan-type Foo int */\nclass Bar {\n    /** @var Foo */\n    private $y;\n}\n",
			"<?php\n/** @phpstan-type Foo int */\nclass Bar {\n    /** @var Foo */\n    private $y;\n}\n",
		},
	})
}
