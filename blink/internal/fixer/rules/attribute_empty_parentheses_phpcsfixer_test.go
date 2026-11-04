package rules

import "testing"

// Cases ported from PHP-CS-Fixer's AttributeEmptyParenthesesFixerTest.
func TestAttributeEmptyParenthesesPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	// default: remove empty parentheses
	aliasRunUpstreamCases(t, AttributeEmptyParentheses{}, []aliasUpstreamCase{
		{"qualified", nil, "<?php\n#[\\A\\B\\Foo()]\nclass S {}", "<?php\n#[\\A\\B\\Foo]\nclass S {}"},
		{"inner whitespace kept", nil, "<?php\n#[\\Bar(      )]\nclass S {}", "<?php\n#[\\Bar      ]\nclass S {}"},
		{"already none", nil, "<?php\n#[Baz]\nclass S {}", "<?php\n#[Baz]\nclass S {}"},
		{"group", nil, "<?php\n#[Bar(), Baz()]\nclass S {}", "<?php\n#[Bar, Baz]\nclass S {}"},
		{"keep arguments", nil, "<?php\n#[Corge(4, 'baz qux')]\nclass S {}", "<?php\n#[Corge(4, 'baz qux')]\nclass S {}"},
		{"mixed group", nil, "<?php\n#[\\A\\B\\Foo(), \\Bar(      ), Baz]\nclass S {}", "<?php\n#[\\A\\B\\Foo, \\Bar      , Baz]\nclass S {}"},
	})
	// use_parentheses true: add empty parentheses
	withParens := map[string]any{"use_parentheses": true}
	aliasRunUpstreamCases(t, AttributeEmptyParentheses{}, []aliasUpstreamCase{
		{"add qualified kept", withParens, "<?php\n#[\\A\\B\\Foo()]\nclass S {}", "<?php\n#[\\A\\B\\Foo()]\nclass S {}"},
		{"add bar", withParens, "<?php\n#[\\Bar]\nclass S {}", "<?php\n#[\\Bar()]\nclass S {}"},
		{"add group", withParens, "<?php\n#[\\Bar, Baz]\nclass S {}", "<?php\n#[\\Bar(), Baz()]\nclass S {}"},
		{"add mixed", withParens, "<?php\n#[\\A\\B\\Foo(), \\Bar, Baz, Corge(4, 'baz qux')]\nclass S {}", "<?php\n#[\\A\\B\\Foo(), \\Bar(), Baz(), Corge(4, 'baz qux')]\nclass S {}"},
	})
}
