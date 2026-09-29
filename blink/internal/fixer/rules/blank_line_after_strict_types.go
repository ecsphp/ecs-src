package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/symplify/coding-standard/blob/main/src/Fixer/Strict/BlankLineAfterStrictTypesFixer.php
//
// BlankLineAfterStrictTypes ensures exactly one blank line after a
// "declare(strict_types=1);" statement.
type BlankLineAfterStrictTypes struct{}

func (BlankLineAfterStrictTypes) Name() string {
	return `Symplify\CodingStandard\Fixer\Strict\BlankLineAfterStrictTypesFixer`
}

func (BlankLineAfterStrictTypes) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Strict/BlankLineAfterStrictTypesFixer.php"
}

func (BlankLineAfterStrictTypes) Fix(s *tokens.Stream) bool {
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Keyword || !strings.EqualFold(s.At(i).Value, "declare") {
			continue
		}
		open := nextSignificantIndex(s, i)
		if open < 0 || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
			continue
		}
		close := s.MatchForward(open)
		if close < 0 {
			continue
		}
		if !declareIsStrictTypes(s, open, close) {
			continue
		}
		semi := nextSignificantIndex(s, close)
		if semi < 0 || s.At(semi).Kind != token.Punct || s.At(semi).Value != ";" {
			continue
		}
		// nothing but the statement's own line may follow before real code
		if semi+1 >= s.Len() {
			return false
		}
		ws := s.At(semi + 1)
		if ws.Kind != token.Whitespace || !strings.Contains(ws.Value, "\n") {
			return false
		}
		// a closing tag or EOF right after needs no blank line
		if semi+2 >= s.Len() || s.At(semi+2).Kind == token.CloseTag {
			return false
		}
		indent := ws.Value[strings.LastIndexByte(ws.Value, '\n')+1:]
		want := "\n\n" + indent
		if ws.Value != want {
			s.SetValue(semi+1, want)
			return true
		}
		return false
	}
	return false
}

func declareIsStrictTypes(s *tokens.Stream, open, close int) bool {
	for k := open + 1; k < close; k++ {
		if s.At(k).Kind == token.Ident && strings.EqualFold(s.At(k).Value, "strict_types") {
			return true
		}
	}
	return false
}
