package rules

import "testing"

// Cases ported from PHP-CS-Fixer's NotOperatorWithSpaceFixerTest; every expected
// output is what PHP-CS-Fixer produces for the input.
func TestNotOperatorWithSpacePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, NotOperatorWithSpace{}, []aliasUpstreamCase{
		{"leading and trailing", nil, "<?php $i = 0; $i++; ++$i; $foo = !false || (!true);", "<?php $i = 0; $i++; ++$i; $foo = ! false || ( ! true);"},
		{"nested", nil, "<?php $i = 0; $i--; --$i; $foo = !false || ($i && !true);", "<?php $i = 0; $i--; --$i; $foo = ! false || ($i && ! true);"},
		{"comment kept", nil, "<?php $i = 0; $i--; $foo = !false || ($i && !/* some comment */true);", "<?php $i = 0; $i--; $foo = ! false || ($i && ! /* some comment */true);"},
		{"existing spaces kept after", nil, "<?php $i = 0; $i--; $foo = !false || ($i && !    true);", "<?php $i = 0; $i--; $foo = ! false || ($i && !    true);"},
		{"existing spaces kept both", nil, "<?php $i = 0; $i--; $foo = !false || ($i &&    !    true);", "<?php $i = 0; $i--; $foo = ! false || ($i &&    !    true);"},
		{"if", nil, "<?php if (!$bar) { echo 1; }", "<?php if ( ! $bar) { echo 1; }"},
	})
}
