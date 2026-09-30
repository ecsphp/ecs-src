package rules

import (
	"slices"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoUnneededControlParenthesesFixer.php
//
// NoUnneededControlParentheses removes a redundant parenthesis pair wrapping the
// whole operand of a control statement: "return ($x);" -> "return $x;". It fires
// for return, echo, print, yield, break, continue, clone and case, only when the
// "(" directly follows the keyword and its matching ")" sits right before the
// statement terminator (";" or, for case, ":"). Nested pairs are peeled in one
// pass. The conservative boundary check leaves anything else (multiple arguments,
// a partial sub-expression, a mid-expression clone) untouched.
type NoUnneededControlParentheses struct {
	// Statements limits the fixed statements; nil means the default set.
	Statements []string
}

func (f NoUnneededControlParentheses) WithConfig(config map[string]any) fixer.Fixer {
	switch list := config["statements"].(type) {
	case []string:
		f.Statements = append([]string{}, list...)
	case []any:
		f.Statements = []string{}
		for _, v := range list {
			if name, ok := v.(string); ok {
				f.Statements = append(f.Statements, name)
			}
		}
	}
	return f
}

func (f NoUnneededControlParentheses) enabled(key string) bool {
	if f.Statements == nil {
		return true
	}
	return slices.Contains(f.Statements, key)
}

func nucpStatementKey(keyword string) string {
	switch keyword {
	case "echo", "print":
		return "echo_print"
	case "case":
		return "switch_case"
	}
	return keyword
}

func (NoUnneededControlParentheses) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\NoUnneededControlParenthesesFixer`
}

func (NoUnneededControlParentheses) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoUnneededControlParenthesesFixer.php"
}

func (f NoUnneededControlParentheses) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword {
			continue
		}
		var term string
		kw := strings.ToLower(t.Value)
		switch kw {
		case "return", "echo", "print", "yield", "break", "continue", "clone":
			term = ";"
		case "case":
			term = ":"
		default:
			continue
		}
		if !f.enabled(nucpStatementKey(kw)) {
			continue
		}
		// a keyword after "->"/"::" is a member name (Foo::case(...)), not a control
		// statement
		if memberPrev(s, i) {
			continue
		}
		// peel redundant pairs: "return (($x));" -> "return $x;"
		for {
			j := nextSignificantIndex(s, i)
			if j < 0 || s.At(j).Kind != token.Punct || s.At(j).Value != "(" {
				break
			}
			k := s.MatchForward(j)
			if k < 0 {
				break
			}
			// the matching ")" must sit right before the terminator, and the pair
			// must not be empty
			nj := nextSignificantIndex(s, k)
			if nj < 0 || s.At(nj).Kind != token.Punct || s.At(nj).Value != term {
				break
			}
			if nextSignificantIndex(s, j) >= k {
				break
			}
			s.RemoveAt(k)
			if j == i+1 {
				// "return($x)" -> keep a separator so the keyword stays a keyword
				s.RemoveAt(j)
				s.InsertAt(j, token.Token{Kind: token.Whitespace, Value: " "})
			} else {
				s.RemoveAt(j)
			}
			changed = true
		}
	}
	return changed
}
