package rules

import "testing"

// Cases ported from PHP-CS-Fixer's NoRedundantReadonlyPropertyFixerTest.
func TestNoRedundantReadonlyPropertyPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, NoRedundantReadonlyProperty{}, []aliasUpstreamCase{
		{"normal properties", nil,
			"<?php\nreadonly class C1\n{\n    private readonly int $bar;\n    private readonly int $baz;\n}\n\nclass C2\n{\n    private readonly int $bar;\n    private readonly int $baz;\n}",
			"<?php\nreadonly class C1\n{\n    private int $bar;\n    private int $baz;\n}\n\nclass C2\n{\n    private readonly int $bar;\n    private readonly int $baz;\n}"},
		{"trait with anonymous class", nil,
			"<?php\ntrait T\n{\n    protected string $foo = 'bar';\n\n    public function setUp(): void\n    {\n        $x = new class() extends \\stdClass {\n            public string $foo;\n        };\n    }\n}",
			"<?php\ntrait T\n{\n    protected string $foo = 'bar';\n\n    public function setUp(): void\n    {\n        $x = new class() extends \\stdClass {\n            public string $foo;\n        };\n    }\n}"},
		{"promoted properties", nil,
			"<?php\nreadonly class C1\n{\n    public function __construct(\n        private readonly int $bar,\n        private readonly int $baz,\n    ) {\n    }\n}\n\nclass C2\n{\n    public function __construct(\n        private readonly int $bar,\n        private readonly int $baz,\n    ) {\n    }\n}",
			"<?php\nreadonly class C1\n{\n    public function __construct(\n        private int $bar,\n        private int $baz,\n    ) {\n    }\n}\n\nclass C2\n{\n    public function __construct(\n        private readonly int $bar,\n        private readonly int $baz,\n    ) {\n    }\n}"},
	})
}
