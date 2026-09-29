package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/NamespaceNotation/SingleBlankLineBeforeNamespaceFixer.php
//
// SingleBlankLineBeforeNamespace ensures exactly one blank line before a
// namespace declaration on its own line (the older sibling of
// BlankLinesBeforeNamespace, kept for parity with configs that reference it).
type SingleBlankLineBeforeNamespace struct{}

func (SingleBlankLineBeforeNamespace) Name() string {
	return `PhpCsFixer\Fixer\NamespaceNotation\SingleBlankLineBeforeNamespaceFixer`
}

func (SingleBlankLineBeforeNamespace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/NamespaceNotation/SingleBlankLineBeforeNamespaceFixer.php"
}

func (SingleBlankLineBeforeNamespace) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "namespace" || memberPrev(s, i) {
			continue
		}
		if i == 0 {
			continue
		}
		prev := s.At(i - 1)
		if prev.Kind == token.Whitespace && hasNewline(prev.Value) && prev.Value != "\n\n" {
			s.SetValue(i-1, "\n\n")
			changed = true
		}
	}
	return changed
}
