package rules

import "testing"

// A single attribute, groups already in alpha order, and a file with no
// attributes are all left untouched.
func TestOrderedAttributesNoop(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, OrderedAttributes{}, []aliasUpstreamCase{
		{"single attribute", nil, "<?php\n#[Foo]\nclass S {}", "<?php\n#[Foo]\nclass S {}"},
		{"already alpha", nil, "<?php\n#[Bar]\n#[Foo]\nclass S {}", "<?php\n#[Bar]\n#[Foo]\nclass S {}"},
		{"no attributes", nil, "<?php\nclass S {}", "<?php\nclass S {}"},
	})
}

// With the default (empty) attributes list, GeneralAttributeRemove removes
// nothing.
func TestGeneralAttributeRemoveDefaultNoop(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, GeneralAttributeRemove{}, []aliasUpstreamCase{
		{"empty config keeps attributes", nil, "<?php\n#[\\A\\B\\Foo]\n#[Bar]\nfunction f() {}\n", "<?php\n#[\\A\\B\\Foo]\n#[Bar]\nfunction f() {}\n"},
		{"explicit empty list", map[string]any{"attributes": []any{}}, "<?php\n#[\\A\\B\\Foo]\nfunction f() {}\n", "<?php\n#[\\A\\B\\Foo]\nfunction f() {}\n"},
	})
}
