package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// TernaryOperatorSpaces ensures a single space around the ternary "?" and ":"
// ("$a?$b:$c" -> "$a ? $b : $c"). The elvis "?:" keeps the "?" and ":"
// adjacent (one space before "?", one after ":").
//
// Like PHP-CS-Fixer, it does not pair a "?" with its ":"; it treats every "?"
// and ":" as ternary unless the tokenizer gives it another role: a nullable
// type "?", a return/enum type colon, a named-argument colon, a switch case
// colon, an alternative-syntax colon or a goto label. "??", "?->" and "::" are
// tokens of their own.
type TernaryOperatorSpaces struct{}

func (TernaryOperatorSpaces) Name() string {
	return `PhpCsFixer\Fixer\Operator\TernaryOperatorSpacesFixer`
}

func (TernaryOperatorSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/TernaryOperatorSpacesFixer.php"
}

func (TernaryOperatorSpaces) Fix(s *tokens.Stream) bool {
	var switchColons map[int]bool
	var ops []int
	for i := 0; i < s.Len(); i++ {
		switch {
		case isPunctVal(s, i, "?"):
			if isNullableType(s, i) {
				continue
			}
		case isPunctVal(s, i, ":"):
			if isTypeColon(s, i) || isNamedArgumentColon(s, i) ||
				belongsToGotoLabel(s, i) || belongsToAlternativeSyntax(s, i) {
				continue
			}
			if switchColons == nil {
				switchColons = switchCaseColons(s)
			}
			if switchColons[i] {
				continue
			}
		default:
			continue
		}
		ops = append(ops, i)
	}

	changed := false
	// Right-to-left so insertions never shift an index still to be visited.
	for k := len(ops) - 1; k >= 0; k-- {
		i := ops[k]
		if s.At(i).Value == "?" {
			if isPunctVal(s, nextSignificantIndex(s, i), ":") {
				// elvis "$a ? : $b": no whitespace between "?" and ":"
				if s.Kind(i+1) == token.Whitespace {
					s.RemoveAt(i + 1)
					changed = true
				}
			} else {
				changed = ternaryEnsureSpace(s, i+1, true) || changed
			}
			changed = ternaryEnsureSpace(s, i-1, false) || changed
			continue
		}
		changed = ternaryEnsureSpace(s, i+1, true) || changed
		if !isPunctVal(s, prevSignificantIndex(s, i), "?") {
			changed = ternaryEnsureSpace(s, i-1, false) || changed
		}
	}
	return changed
}

// ternaryEnsureSpace mirrors TernaryOperatorSpacesFixer::ensureWhitespaceExistence:
// whitespace at index becomes a single space unless it holds a newline or
// follows a comment; missing whitespace is inserted after (after) or before the
// operator.
func ternaryEnsureSpace(s *tokens.Stream, index int, after bool) bool {
	if index < 0 || index >= s.Len() {
		return false
	}
	if s.Kind(index) == token.Whitespace {
		v := s.At(index).Value
		if hasNewline(v) || v == " " {
			return false
		}
		if index > 0 && (s.Kind(index-1) == token.Comment || s.Kind(index-1) == token.DocComment) {
			return false
		}
		s.SetValue(index, " ")
		return true
	}
	if !after {
		index++
	}
	s.InsertAt(index, token.Token{Kind: token.Whitespace, Value: " "})
	return true
}
