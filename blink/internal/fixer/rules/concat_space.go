package rules

import (
	"strings"

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
	changed := false
	if !f.None {
		changed = normalizeSpaceAround(s, isConcat)
	} else {
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
	}
	// Attributes are lexed as one opaque token, so the loop above never sees the
	// "." inside them; normalize their concat spacing by rewriting the token text.
	for j := 0; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.Comment || !strings.HasPrefix(t.Value, "#[") {
			continue
		}
		if nv, ch := spaceAttributeConcat(t.Value, f.None); ch {
			s.SetValue(j, nv)
			changed = true
		}
	}
	return changed
}

// spaceAttributeConcat normalizes same-line spacing around each top-level "."
// concat operator in an attribute token's text. Dots inside string literals,
// "..." spreads and float literals are left alone, and newlines are preserved.
func spaceAttributeConcat(value string, none bool) (string, bool) {
	var b []byte
	changed := false
	var quote byte
	for i := 0; i < len(value); i++ {
		c := value[i]
		if quote != 0 {
			b = append(b, c)
			if c == '\\' && i+1 < len(value) {
				i++
				b = append(b, value[i])
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '<' && strings.HasPrefix(value[i:], "<<<") {
			if end := heredocEnd(value, i); end > i {
				b = append(b, value[i:end]...)
				i = end - 1
				continue
			}
		}
		if c == '\'' || c == '"' {
			quote = c
			b = append(b, c)
			continue
		}
		if c != '.' || !isConcatDot(value, i) {
			b = append(b, c)
			continue
		}

		// trailing same-line whitespace already written before the "."
		start := len(b)
		for start > 0 && (b[start-1] == ' ' || b[start-1] == '\t') {
			start--
		}
		leadNewline := start > 0 && (b[start-1] == '\n' || b[start-1] == '\r')
		// leading same-line whitespace after the "."
		j := i + 1
		for j < len(value) && (value[j] == ' ' || value[j] == '\t') {
			j++
		}
		trailNewline := j >= len(value) || value[j] == '\n' || value[j] == '\r'

		if none {
			if start > 0 && !leadNewline && len(b)-start > 0 {
				if isDigitByte(b[start-1]) { // guard against number.number re-lexing
					if len(b)-start != 1 {
						b = append(b[:start], ' ')
						changed = true
					}
				} else {
					b = b[:start]
					changed = true
				}
			}
			b = append(b, '.')
			if !trailNewline && j > i+1 {
				if isDigitByte(value[j]) {
					b = append(b, ' ')
					if j-(i+1) != 1 {
						changed = true
					}
				} else {
					changed = true
				}
				i = j - 1
			}
			continue
		}

		// none == false: force a single same-line space on each side
		if start > 0 && !leadNewline {
			if len(b)-start != 1 || b[start] != ' ' {
				b = append(b[:start], ' ')
				changed = true
			}
		}
		b = append(b, '.')
		if !trailNewline {
			b = append(b, ' ')
			if j-(i+1) != 1 {
				changed = true
			}
			i = j - 1
		}
	}
	return string(b), changed
}

// isConcatDot reports whether the "." at index i is a concatenation operator,
// not part of a "..." spread or a float literal (e.g. 1.5, .5, 1.).
func isConcatDot(value string, i int) bool {
	if i+1 < len(value) && value[i+1] == '.' {
		return false
	}
	if i > 0 && value[i-1] == '.' {
		return false
	}
	if i > 0 && isDigitByte(value[i-1]) {
		k := i - 1
		for k > 0 && isDigitByte(value[k-1]) {
			k--
		}
		// a digit run not preceded by a name char is a number -> "." is a float dot
		if k == 0 || !isIdentByte(value[k-1]) {
			return false
		}
	}
	if i+1 < len(value) && isDigitByte(value[i+1]) && (i == 0 || !isDigitByte(value[i-1])) {
		p := i - 1
		for p >= 0 && (value[p] == ' ' || value[p] == '\t') {
			p--
		}
		// ".5" is a float unless it follows an operand (identifier, string, ) or ])
		if p < 0 {
			return false
		}
		c := value[p]
		if !isIdentByte(c) && c != '\'' && c != '"' && c != ')' && c != ']' {
			return false
		}
	}
	return true
}

func isDigitByte(c byte) bool { return c >= '0' && c <= '9' }

// concatSpaceKeepsSpace reports whether the operand at idx must stay spaced from
// "." (a number would lex as a float, a comment would swallow the operator).
func concatSpaceKeepsSpace(s *tokens.Stream, idx int) bool {
	if idx < 0 {
		return false
	}
	k := s.At(idx).Kind
	return k == token.Number || k == token.Comment || k == token.DocComment
}
