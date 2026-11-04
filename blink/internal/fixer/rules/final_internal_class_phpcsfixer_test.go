package rules

import "testing"

// Cases ported from PHP-CS-Fixer's FinalInternalClassFixerTest.
func TestFinalInternalClassPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	custom := map[string]any{
		"include": []any{"@Custom"},
		"exclude": []any{"@not-fix"},
	}
	aliasRunUpstreamCases(t, FinalInternalClass{}, []aliasUpstreamCase{
		{"internal multiline", nil,
			"<?php\n/**\n * @internal\n */\nclass Sample\n{\n}\n",
			"<?php\n/**\n * @internal\n */\nfinal class Sample\n{\n}\n"},
		{"internal single line", nil,
			"<?php /** @internal */ class Sample {}",
			"<?php /** @internal */ final class Sample {}"},
		{"no docblock not fixed", nil, "<?php class Sample {}", "<?php class Sample {}"},
		{"abstract not fixed", nil, "<?php /** @internal */ abstract class Sample {}", "<?php /** @internal */ abstract class Sample {}"},
		{"already final", nil, "<?php /** @internal */ final class Sample {}", "<?php /** @internal */ final class Sample {}"},
		{"other annotation not fixed", nil, "<?php /** @codeCoverageIgnore */ class Sample {}", "<?php /** @codeCoverageIgnore */ class Sample {}"},
		{"internal attribute", nil, "<?php #[Internal] class Sample {}", "<?php #[Internal] final class Sample {}"},
		{"custom include", custom,
			"<?php\n/**\n * @CUSTOM\n */\nclass A{}\n",
			"<?php\n/**\n * @CUSTOM\n */\nfinal class A{}\n"},
		{"custom include with excluded", custom,
			"<?php\n/**\n * @CUSTOM\n * @not-fix\n */\nclass B{}\n",
			"<?php\n/**\n * @CUSTOM\n * @not-fix\n */\nclass B{}\n"},
	})
}
