package rules

import "testing"

// Cases ported from PHP-CS-Fixer's MultilinePromotedPropertiesFixerTest
// (tests/Fixer/FunctionNotation); every expected output is what PHP-CS-Fixer
// produces for the input.
func TestMultilinePromotedPropertiesPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, MultilinePromotedProperties{}, []aliasUpstreamCase{
		{"single parameter", nil,
			`<?php class Foo {
                public function __construct(public int $x) {}
            }`,
			`<?php class Foo {
                public function __construct(
                    public int $x
                ) {}
            }`},
		{"single parameter with trailing comma", nil,
			`<?php class Foo {
                public function __construct(protected int $x,) {}
            }`,
			`<?php class Foo {
                public function __construct(
                    protected int $x,
                ) {}
            }`},
		{"multiple parameters", nil,
			`<?php class Foo {
                public function __construct(private int $x, private int $y, private int $z) {}
            }`,
			`<?php class Foo {
                public function __construct(
                    private int $x,
                    private int $y,
                    private int $z
                ) {}
            }`},
		{"multiple parameters and only one promoted", nil,
			`<?php class Foo {
                public function __construct(int $x, private int $y, int $z) {}
            }`,
			`<?php class Foo {
                public function __construct(
                    int $x,
                    private int $y,
                    int $z
                ) {}
            }`},
		{"parameters with default values", nil,
			`<?php class Foo {
                public function __construct(private array $a = [1, 2, 3, 4], private bool $b = self::DEFAULT_B,) {}
            }`,
			`<?php class Foo {
                public function __construct(
                    private array $a = [1, 2, 3, 4],
                    private bool $b = self::DEFAULT_B,
                ) {}
            }`},
		{"parameters with attributes", nil,
			`<?php class Foo {
                public function __construct(private array $a = [1, 2, 3, 4], #[Bar(1, 2, 3)] private bool $b = self::DEFAULT_B,) {}
            }`,
			`<?php class Foo {
                public function __construct(
                    private array $a = [1, 2, 3, 4],
                    #[Bar(1, 2, 3)] private bool $b = self::DEFAULT_B,
                ) {}
            }`},
		{"multiple classes", nil,
			`<?php
            class ClassWithSinglePromotedProperty {
                public function __construct(private int $foo) {}
            }
            class ClassWithoutConstructor {}
            class ClassWithoutPromotedProperties {
                public function __construct(string $a, string $b) {}
            }
            class ClassWithMultiplePromotedProperties {
                public function __construct(private int $x, private int $y, private int $z) {}
            }`,
			`<?php
            class ClassWithSinglePromotedProperty {
                public function __construct(
                    private int $foo
                ) {}
            }
            class ClassWithoutConstructor {}
            class ClassWithoutPromotedProperties {
                public function __construct(string $a, string $b) {}
            }
            class ClassWithMultiplePromotedProperties {
                public function __construct(
                    private int $x,
                    private int $y,
                    private int $z
                ) {}
            }`},
		{"0 parameters with 0 configured", map[string]any{"minimum_number_of_parameters": 0},
			`<?php class Foo {
                    public function __construct() {}
                }`,
			`<?php class Foo {
                    public function __construct() {}
                }`},

		// foreach ([0, 1, 2]): 2 parameters fixed to multiline
		{"2 parameters with 0 configured", map[string]any{"minimum_number_of_parameters": 0},
			`<?php class Foo {
                    public function __construct(private int $x, private int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(
                        private int $x,
                        private int $y
                    ) {}
                }`},
		{"2 parameters with 1 configured", map[string]any{"minimum_number_of_parameters": 1},
			`<?php class Foo {
                    public function __construct(private int $x, private int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(
                        private int $x,
                        private int $y
                    ) {}
                }`},
		{"2 parameters with 2 configured", map[string]any{"minimum_number_of_parameters": 2},
			`<?php class Foo {
                    public function __construct(private int $x, private int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(
                        private int $x,
                        private int $y
                    ) {}
                }`},
		{"2 parameters and only one promoted with 0 configured", map[string]any{"minimum_number_of_parameters": 0},
			`<?php class Foo {
                    public function __construct(int $x, private int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(
                        int $x,
                        private int $y
                    ) {}
                }`},
		{"2 parameters and only one promoted with 1 configured", map[string]any{"minimum_number_of_parameters": 1},
			`<?php class Foo {
                    public function __construct(int $x, private int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(
                        int $x,
                        private int $y
                    ) {}
                }`},
		{"2 parameters and only one promoted with 2 configured", map[string]any{"minimum_number_of_parameters": 2},
			`<?php class Foo {
                    public function __construct(int $x, private int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(
                        int $x,
                        private int $y
                    ) {}
                }`},

		// foreach ([3, 4]): 2 parameters left unchanged (below threshold, already single line)
		{"2 parameters with 3 configured", map[string]any{"minimum_number_of_parameters": 3},
			`<?php class Foo {
                    public function __construct(private int $x, private int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(private int $x, private int $y) {}
                }`},
		{"2 parameters with 4 configured", map[string]any{"minimum_number_of_parameters": 4},
			`<?php class Foo {
                    public function __construct(private int $x, private int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(private int $x, private int $y) {}
                }`},
		{"2 parameters and only one promoted with 3 configured", map[string]any{"minimum_number_of_parameters": 3},
			`<?php class Foo {
                    public function __construct(int $x, private int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(int $x, private int $y) {}
                }`},
		{"2 parameters and only one promoted with 4 configured", map[string]any{"minimum_number_of_parameters": 4},
			`<?php class Foo {
                    public function __construct(int $x, private int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(int $x, private int $y) {}
                }`},

		// foreach ([1, 2, 3, 4]): no promoted parameter, never touched
		{"2 parameters and none promoted with 1 configured", map[string]any{"minimum_number_of_parameters": 1},
			`<?php class Foo {
                    public function __construct(int $x, int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(int $x, int $y) {}
                }`},
		{"2 parameters and none promoted with 2 configured", map[string]any{"minimum_number_of_parameters": 2},
			`<?php class Foo {
                    public function __construct(int $x, int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(int $x, int $y) {}
                }`},
		{"2 parameters and none promoted with 3 configured", map[string]any{"minimum_number_of_parameters": 3},
			`<?php class Foo {
                    public function __construct(int $x, int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(int $x, int $y) {}
                }`},
		{"2 parameters and none promoted with 4 configured", map[string]any{"minimum_number_of_parameters": 4},
			`<?php class Foo {
                    public function __construct(int $x, int $y) {}
                }`,
			`<?php class Foo {
                    public function __construct(int $x, int $y) {}
                }`},

		{"single parameter unwrapped with minimum 2", map[string]any{"minimum_number_of_parameters": 2},
			`<?php class Foo {
                public function __construct(
                    private int $x
                ) {}
            }`,
			`<?php class Foo {
                public function __construct(private int $x) {}
            }`},
		{"single parameter with trailing comma unwrapped with minimum 2", map[string]any{"minimum_number_of_parameters": 2},
			`<?php class Foo {
                public function __construct(
                    private int $x,
                ) {}
            }`,
			`<?php class Foo {
                public function __construct(private int $x,) {}
            }`},
		{"single non-promoted parameter not unwrapped with minimum 2", map[string]any{"minimum_number_of_parameters": 2},
			`<?php class Foo {
                public function __construct(
                    int $x
                ) {}
            }`,
			`<?php class Foo {
                public function __construct(
                    int $x
                ) {}
            }`},
		{"single parameter with comment not unwrapped with minimum 2", map[string]any{"minimum_number_of_parameters": 2},
			`<?php class Foo {
                public function __construct(
                    // The name
                    private string $name,
                ) {}
            }`,
			`<?php class Foo {
                public function __construct(
                    // The name
                    private string $name,
                ) {}
            }`},
		{"single parameter with attribute not unwrapped with minimum 2", map[string]any{"minimum_number_of_parameters": 2},
			`<?php class Foo {
                public function __construct(
                    #[Assert\NotBlank]
                    private string $name,
                ) {}
            }`,
			`<?php class Foo {
                public function __construct(
                    #[Assert\NotBlank]
                    private string $name,
                ) {}
            }`},
		{"two parameters unwrapped with minimum 3", map[string]any{"minimum_number_of_parameters": 3},
			`<?php class Foo {
                public function __construct(
                    private int $x,
                    private int $y
                ) {}
            }`,
			`<?php class Foo {
                public function __construct(private int $x, private int $y) {}
            }`},
		{"blank lines removed", nil,
			`<?php class Foo {
                public function __construct(
                    private int $x,

                    private int $y,

                    private int $z
                ) {}
            }`,
			`<?php class Foo {
                public function __construct(
                    private int $x,
                    private int $y,
                    private int $z
                ) {}
            }`},
		{"no blank lines", nil,
			`<?php class Foo {
                public function __construct(
                    private int $x, private int $y,

                    private int $z
                ) {}
            }`,
			`<?php class Foo {
                public function __construct(
                    private int $x,
                    private int $y,
                    private int $z
                ) {}
            }`},
		{"blank lines kept", map[string]any{"keep_blank_lines": true},
			`<?php class Foo {
                public function __construct(
                    private int $x, private int $y,

                    private int $z
                ) {}
            }`,
			`<?php class Foo {
                public function __construct(
                    private int $x,
                    private int $y,

                    private int $z
                ) {}
            }`},

		// provideFix81Cases (new-in-initializers)
		{"new in initializers", nil,
			"<?php class Foo\n{\n    public function __construct(public Bar $x = new Bar(), public Bar $y = new Bar())\n    {\n    }\n}",
			"<?php class Foo\n{\n    public function __construct(\n        public Bar $x = new Bar(),\n        public Bar $y = new Bar()\n    )\n    {\n    }\n}"},
		{"new in initializers with trailing comma", nil,
			"<?php class Foo\n{\n    public function __construct(public Bar $x = new Bar(), public Bar $y = new Bar(),)\n    {\n    }\n}",
			"<?php class Foo\n{\n    public function __construct(\n        public Bar $x = new Bar(),\n        public Bar $y = new Bar(),\n    )\n    {\n    }\n}"},
	})
}
