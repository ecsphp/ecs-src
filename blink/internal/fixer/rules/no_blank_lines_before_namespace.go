package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/NamespaceNotation/NoBlankLinesBeforeNamespaceFixer.php
//
// NoBlankLinesBeforeNamespace removes blank lines before a namespace
// declaration. It is the deprecated proxy for blank_lines_before_namespace with
// min_line_breaks 0 and max_line_breaks 1: the line breaks before the namespace
// keyword are clamped to at most one, and never added when there are none.
type NoBlankLinesBeforeNamespace struct{}

func (NoBlankLinesBeforeNamespace) Name() string {
	return `PhpCsFixer\Fixer\NamespaceNotation\NoBlankLinesBeforeNamespaceFixer`
}

func (NoBlankLinesBeforeNamespace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/NamespaceNotation/NoBlankLinesBeforeNamespaceFixer.php"
}

func (NoBlankLinesBeforeNamespace) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "namespace" || memberPrev(s, i) {
			continue
		}
		if aliasIsNamespaceOperator(s, i) || i == 0 {
			continue
		}
		prev := s.At(i - 1)
		if prev.Kind != token.Whitespace || !hasNewline(prev.Value) {
			continue
		}
		lineEnding := "\n"
		if strings.Contains(prev.Value, "\r\n") {
			lineEnding = "\r\n"
		}
		target := clampLineBreaks(strings.Count(prev.Value, "\n"), 0, 1)
		indent := prev.Value
		if idx := strings.LastIndexByte(prev.Value, '\n'); idx >= 0 {
			indent = prev.Value[idx+1:]
		}
		want := strings.Repeat(lineEnding, target) + indent
		if prev.Value != want {
			s.SetValue(i-1, want)
			changed = true
		}
	}
	return changed
}
