package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/MethodChainingIndentationFixer.php
//
// MethodChainingIndentation aligns a chained "->"/"?->" that already sits at the
// start of a line to one indentation level (four spaces) past the line the chain
// starts on. It only adjusts the indentation of an existing line break, so it
// never joins or splits calls and is a no-op once the chain is aligned.
type MethodChainingIndentation struct{}

func (MethodChainingIndentation) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\MethodChainingIndentationFixer`
}

func (MethodChainingIndentation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/MethodChainingIndentationFixer.php"
}

func (MethodChainingIndentation) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 1; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || (t.Value != "->" && t.Value != "?->") {
			continue
		}
		ws := s.At(i - 1)
		if ws.Kind != token.Whitespace || !strings.Contains(ws.Value, "\n") {
			continue // only operators that already begin a line
		}
		want := chainFirstLineIndent(s, i) + "    "
		nl := strings.LastIndexByte(ws.Value, '\n')
		got := ws.Value[nl+1:]
		if got == want {
			continue
		}
		s.SetValue(i-1, ws.Value[:nl+1]+want)
		changed = true
	}
	return changed
}
