package rules

import (
	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/ConcatSpaceFixer.php
//
// ConcatSpace forces a single space around the "." concatenation operator
// (spacing: one): "'a'.'b'" becomes "'a' . 'b'". With None set (spacing: none)
// the spaces are removed instead. Whitespace across a newline is preserved.
type ConcatSpace struct {
	None bool
}

func (ConcatSpace) Name() string {
	return `PhpCsFixer\Fixer\Operator\ConcatSpaceFixer`
}

func (ConcatSpace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/ConcatSpaceFixer.php"
}

func (f ConcatSpace) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["spacing"].(string); ok {
		f.None = v == "none"
	}
	return f
}

func (f ConcatSpace) Fix(s *tokens.Stream) bool {
	isConcat := func(s *tokens.Stream, i int) bool {
		t := s.At(i)
		return t.Kind == token.Punct && t.Value == "."
	}
	if !f.None {
		return normalizeSpaceAround(s, isConcat)
	}
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		if !isConcat(s, i) {
			continue
		}
		if i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) &&
			!concatSpaceKeepsSpace(s, nextSignificantIndex(s, i)) {
			s.RemoveAt(i + 1)
			changed = true
		}
		if i >= 1 && s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) &&
			!concatSpaceKeepsSpace(s, prevSignificantIndex(s, i)) {
			s.RemoveAt(i - 1)
			changed = true
		}
	}
	return changed
}

// concatSpaceKeepsSpace reports whether the operand at idx must stay spaced from
// "." (a number would lex as a float, a comment would swallow the operator).
func concatSpaceKeepsSpace(s *tokens.Stream, idx int) bool {
	if idx < 0 {
		return false
	}
	k := s.At(idx).Kind
	return k == token.Number || k == token.Comment || k == token.DocComment
}
