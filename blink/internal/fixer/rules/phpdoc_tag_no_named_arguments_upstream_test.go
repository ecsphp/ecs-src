package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (PhpdocTagNoNamedArgumentsFixer).
func TestPhpdocTagNoNamedArgumentsPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpdocTagNoNamedArguments{}, []aliasUpstreamCase{
		{
			"basic class", nil,
			"<?php\nclass Foo\n{\n}\n",
			"<?php\n\n/**\n * @no-named-arguments\n */\nclass Foo\n{\n}\n",
		},
		{
			"interface", nil,
			"<?php\ninterface Foo\n{\n}\n",
			"<?php\n\n/**\n * @no-named-arguments\n */\ninterface Foo\n{\n}\n",
		},
		{
			"trait", nil,
			"<?php\ntrait Foo\n{\n}\n",
			"<?php\n\n/**\n * @no-named-arguments\n */\ntrait Foo\n{\n}\n",
		},
		{
			"enum", nil,
			"<?php\nenum Foo\n{\n}\n",
			"<?php\n\n/**\n * @no-named-arguments\n */\nenum Foo\n{\n}\n",
		},
		{
			"existing docblock gets tag", nil,
			"<?php\n/**\n * Something.\n */\nclass Foo\n{\n}\n",
			"<?php\n/**\n * Something.\n * @no-named-arguments\n */\nclass Foo\n{\n}\n",
		},
		{
			"class already tagged is untouched", nil,
			"<?php\n/**\n * @no-named-arguments\n */\nclass Foo\n{\n}\n",
			"<?php\n/**\n * @no-named-arguments\n */\nclass Foo\n{\n}\n",
		},
		{
			"description added", map[string]any{"description": "Parameter names are not covered by the backward compatibility promise."},
			"<?php\nclass Foo\n{\n}\n",
			"<?php\n\n/**\n * @no-named-arguments Parameter names are not covered by the backward compatibility promise.\n */\nclass Foo\n{\n}\n",
		},
		{
			"description normalizes existing tag", map[string]any{"description": "BC break."},
			"<?php\n/**\n * @no-named-arguments old text\n */\nclass Foo\n{\n}\n",
			"<?php\n/**\n * @no-named-arguments BC break.\n */\nclass Foo\n{\n}\n",
		},
		{
			"anonymous class skipped", nil,
			"<?php\n$x = new class {\n};\n",
			"<?php\n$x = new class {\n};\n",
		},
		{
			"internal class skipped when fix_internal false", map[string]any{"fix_internal": false},
			"<?php\n/**\n * @internal\n */\nclass Foo\n{\n}\n",
			"<?php\n/**\n * @internal\n */\nclass Foo\n{\n}\n",
		},
		{
			"internal class fixed by default", nil,
			"<?php\n/**\n * @internal\n */\nclass Foo\n{\n}\n",
			"<?php\n/**\n * @internal\n * @no-named-arguments\n */\nclass Foo\n{\n}\n",
		},
		{
			"attribute class skipped when fix_attribute false", map[string]any{"fix_attribute": false},
			"<?php\n#[Attribute]\nclass Foo\n{\n}\n",
			"<?php\n#[Attribute]\nclass Foo\n{\n}\n",
		},
		{
			"attribute class fixed by default", nil,
			"<?php\n#[Attribute]\nclass Foo\n{\n}\n",
			"<?php\n\n/**\n * @no-named-arguments\n */\n#[Attribute]\nclass Foo\n{\n}\n",
		},
		{
			"single line docblock made multiline", nil,
			"<?php\n/** Something. */\nclass Foo\n{\n}\n",
			"<?php\n/**\n * Something.\n * @no-named-arguments\n */\nclass Foo\n{\n}\n",
		},
	})
}
