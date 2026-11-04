package rules

import "testing"

// Cases ported from PHP-CS-Fixer's NoBlankLinesBeforeNamespaceFixerTest.
func TestNoBlankLinesBeforeNamespacePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, NoBlankLinesBeforeNamespace{}, []aliasUpstreamCase{
		{"inline", nil, "<?php namespace Some\\Name\\Space;", "<?php namespace Some\\Name\\Space;"},
		{"single newline", nil, "<?php\nnamespace X;", "<?php\nnamespace X;"},
		{"collapse blanks", nil, "<?php\n\n\n\nnamespace X;", "<?php\nnamespace X;"},
		{"crlf single", nil, "<?php\r\nnamespace X;", "<?php\r\nnamespace X;"},
		{"crlf collapse", nil, "<?php\r\n\r\n\r\n\r\nnamespace X;", "<?php\r\nnamespace X;"},
		{"namespace operator", nil, "<?php\n\nnamespace\\Sub\\Foo::bar();", "<?php\n\nnamespace\\Sub\\Foo::bar();"},
		{"comment keeps indent", nil, "<?php\n    // Foo\n    \n    namespace Foo;", "<?php\n    // Foo\n    namespace Foo;"},
		{"comment no indent", nil, "<?php\n// Foo\n    \nnamespace Foo;", "<?php\n// Foo\nnamespace Foo;"},
	})
}
