package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/TrailingCommaInMultilineFixer.php
//
// TrailingCommaInMultiline adds a trailing comma to the last element of a
// multi-line array literal (the fixer default: elements = ['arrays']).
type TrailingCommaInMultiline struct{}

func (TrailingCommaInMultiline) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\TrailingCommaInMultilineFixer`
}

func (TrailingCommaInMultiline) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/TrailingCommaInMultilineFixer.php"
}

func (TrailingCommaInMultiline) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || t.Value != "]" {
			continue
		}
		open := s.MatchBackward(i)
		if open < 0 || isOffsetOpen(s, open) {
			continue // offset access, not an array literal
		}

		// last real element before the close (skip trailing whitespace/comments)
		p := i - 1
		for p > open && (s.At(p).Kind == token.Whitespace ||
			s.At(p).Kind == token.Comment || s.At(p).Kind == token.DocComment) {
			p--
		}
		if p <= open {
			continue // empty
		}
		if s.At(p).Value == "," || s.At(p).Value == "..." {
			continue // already trailing, or a spread (no trailing comma after it)
		}
		// multi-line only: a newline sits between the last element and the close
		multiline := false
		for k := p + 1; k < i; k++ {
			if s.At(k).Kind == token.Whitespace && hasNewline(s.At(k).Value) {
				multiline = true
				break
			}
		}
		if !multiline {
			continue
		}
		s.InsertAt(p+1, token.Token{Kind: token.Punct, Value: ","})
		changed = true
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/NoTrailingCommaInSinglelineFixer.php
//
// NoTrailingCommaInSingleline removes a trailing comma before a single-line
// closing ")", "]" or "}".
type NoTrailingCommaInSingleline struct{}

func (NoTrailingCommaInSingleline) Name() string {
	return `PhpCsFixer\Fixer\Basic\NoTrailingCommaInSinglelineFixer`
}

func (NoTrailingCommaInSingleline) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/NoTrailingCommaInSinglelineFixer.php"
}

func (NoTrailingCommaInSingleline) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "," {
			continue
		}
		j := i + 1
		for j < s.Len() && s.At(j).Kind == token.Whitespace {
			if hasNewline(s.At(j).Value) {
				j = -1
				break
			}
			j++
		}
		if j < 0 || j >= s.Len() {
			continue
		}
		if v := s.At(j).Value; v == ")" || v == "]" || v == "}" {
			s.RemoveAt(i)
			i--
			changed = true
		}
	}
	return changed
}
