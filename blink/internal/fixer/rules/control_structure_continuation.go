package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/ControlStructureContinuationPositionFixer.php
//
// ControlStructureContinuationPosition puts a control-structure continuation
// keyword on the same line as the preceding "}" (the default "same_line"):
// "}\nelse {" becomes "} else {". It covers else/elseif/catch/finally and the
// "while" of a do-while.
type ControlStructureContinuationPosition struct{}

func (ControlStructureContinuationPosition) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\ControlStructureContinuationPositionFixer`
}

func (ControlStructureContinuationPosition) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/ControlStructureContinuationPositionFixer.php"
}

func (ControlStructureContinuationPosition) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword {
			continue
		}
		lw := strings.ToLower(t.Value)
		cont := lw == "else" || lw == "elseif" || lw == "catch" || lw == "finally" || lw == "while"
		if !cont {
			continue
		}
		// the keyword must directly follow a "}" with a single whitespace token
		// carrying a newline (nothing to do when already on the same line, and a
		// comment in between is left alone)
		pi := prevSignificantIndex(s, i)
		if pi < 0 || i != pi+2 {
			continue
		}
		if s.At(pi).Kind != token.Punct || s.At(pi).Value != "}" {
			continue
		}
		ws := s.At(pi + 1)
		if ws.Kind != token.Whitespace || !hasNewline(ws.Value) {
			continue
		}
		// a bare "while" is a loop header, not a continuation; only the "while" of
		// a do-while (whose "}" closes a "do" block) moves up
		if lw == "while" && !closesDoBlock(s, pi) {
			continue
		}
		s.SetValue(pi+1, " ")
		changed = true
	}
	return changed
}

// closesDoBlock reports whether the "}" at idx closes the body of a "do" block.
func closesDoBlock(s *tokens.Stream, idx int) bool {
	open := s.MatchBackward(idx)
	if open < 0 {
		return false
	}
	kind, kw := classifyBrace(s, open)
	return kind == braceControl && kw >= 0 && strings.ToLower(s.At(kw).Value) == "do"
}
