package rules

import "testing"

// Cases ported from PHP-CS-Fixer's StrictComparisonFixerTest.
func TestStrictComparisonPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, StrictComparison{}, []aliasUpstreamCase{
		{"equal", nil, "<?php $a == $b;", "<?php $a === $b;"},
		{"not equal", nil, "<?php $a != $b;", "<?php $a !== $b;"},
		{"angle not equal", nil, "<?php $a <> $b;", "<?php $a !== $b;"},
		{"inside string", nil, `<?php echo "$a === $b";`, `<?php echo "$a === $b";`},
	})
}
