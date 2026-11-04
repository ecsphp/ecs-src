package rules

import "testing"

// Cases ported from PHP-CS-Fixer's ReturnToYieldFromFixerTest; every expected
// output is what PHP-CS-Fixer produces for the input.
func TestReturnToYieldFromPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, ReturnToYieldFrom{}, []aliasUpstreamCase{
		{"closure in array, no change", nil, "<?php return [function() { return [1, 2, 3]; }];", "<?php return [function() { return [1, 2, 3]; }];"},
		{"arrow fn in array, no change", nil, "<?php return [fn() => [1, 2, 3]];", "<?php return [fn() => [1, 2, 3]];"},
		{"return non-array, no change", nil, "<?php function foo(): iterable { return $z; }\n\nreturn [1,2] ?>  X  <?php { echo 2; }", "<?php function foo(): iterable { return $z; }\n\nreturn [1,2] ?>  X  <?php { echo 2; }"},
		{"no return type, no change", nil, "<?php function foo() { return [1, 2, 3]; }", "<?php function foo() { return [1, 2, 3]; }"},
		{"non-iterable type, no change", nil, "<?php function foo(): MyAwesomeIterableType { return [1, 2, 3]; }", "<?php function foo(): MyAwesomeIterableType { return [1, 2, 3]; }"},
		{"nested braces, no change", nil, "<?php function foo(): iterable { if (true) { return [1]; } else { return [2]; } }", "<?php function foo(): iterable { if (true) { return [1]; } else { return [2]; } }"},
		{"nullable iterable, no change", nil, "<?php function foo(): ?iterable { return [1, 2, 3]; }", "<?php function foo(): ?iterable { return [1, 2, 3]; }"},
		{"closure with iterable", nil, "<?php return [function(): iterable { return [1, 2, 3]; }];", "<?php return [function(): iterable { yield from [1, 2, 3]; }];"},
		{"method", nil, "<?php class Foo {\n                function bar(): iterable { return [1, 2, 3]; }\n            }", "<?php class Foo {\n                function bar(): iterable { yield from [1, 2, 3]; }\n            }"},
		{"trailing semicolons", nil, "<?php function foo(): iterable { return [1, 2, 3];;;;;;;; }", "<?php function foo(): iterable { yield from [1, 2, 3];;;;;;;; }"},
		{"array() syntax", nil, "<?php function foo(): iterable { return array(1, 2, 3); }", "<?php function foo(): iterable { yield from array(1, 2, 3); }"},
		{"statement before", nil, "<?php function foo(): iterable { $x = 0; return [1, 2, 3]; }", "<?php function foo(): iterable { $x = 0; yield from [1, 2, 3]; }"},
		{"uppercase iterable", nil, "<?php function foo(): ITERABLE { return [1, 2, 3]; }", "<?php function foo(): ITERABLE { yield from [1, 2, 3]; }"},
		{"closure assignment", nil, "<?php $f = function(): iterable { return [1, 2, 3]; };", "<?php $f = function(): iterable { yield from [1, 2, 3]; };"},
		{"mixed functions", nil, "<?php\n                function foo(): array { return [3, 4]; }\n                function bar(): iterable { return [1, 2]; }\n                function baz(): int { return 5; }\n            ", "<?php\n                function foo(): array { return [3, 4]; }\n                function bar(): iterable { yield from [1, 2]; }\n                function baz(): int { return 5; }\n            "},
	})
}
