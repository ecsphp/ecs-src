package rules

import "testing"

// Cases ported from PHP-CS-Fixer's PhpdocReadonlyClassCommentToKeywordFixerTest.
func TestPhpdocReadonlyClassCommentToKeywordPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpdocReadonlyClassCommentToKeyword{}, []aliasUpstreamCase{
		{"single line comment", nil,
			"<?php\n\n/** @readonly */\nclass C {\n}",
			"<?php\n\n\nreadonly class C {\n}"},
		{"only annotation", nil,
			"<?php\n\n/**\n * @readonly\n */\nclass C {\n}",
			"<?php\n\n/**\n */\nreadonly class C {\n}"},
		{"keep description", nil,
			"<?php\n\n/**\n * Very impressive class\n *\n * @readonly\n */\nclass C {\n}",
			"<?php\n\n/**\n * Very impressive class\n *\n */\nreadonly class C {\n}"},
		{"final class", nil,
			"<?php\n\n/**\n * @readonly\n */\nfinal class C {\n}",
			"<?php\n\n/**\n */\nfinal readonly class C {\n}"},
		{"abstract class", nil,
			"<?php\n\n/**\n * @readonly\n */\nabstract class C {\n}",
			"<?php\n\n/**\n */\nabstract readonly class C {\n}"},
		{"already readonly", nil,
			"<?php\n/**\n * @readonly\n */\nreadonly class A {\n}",
			"<?php\n/**\n */\nreadonly class A {\n}"},
		{"no readonly annotation", nil,
			"<?php\n/** Class A. */\nclass A {\n}",
			"<?php\n/** Class A. */\nclass A {\n}"},
	})
}
