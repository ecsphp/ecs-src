package rules

import (
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/IsNullFixer.php
//
// IsNull rewrites is_null($x) to "null === $x" and "! is_null($x)" to
// "null !== $x". The argument keeps its parentheses only when it contains a
// top-level operator (e.g. an assignment), where dropping them would change
// precedence.
type IsNull struct{}

func (IsNull) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\IsNullFixer`
}

func (IsNull) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/IsNullFixer.php"
}

func (IsNull) Fix(s *tokens.Stream) bool {
	type site struct {
		start, argOpen, argClose int
		neg                      bool
	}
	var sites []site
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Ident || !strings.EqualFold(t.Value, "is_null") {
			continue
		}
		pj := prevSignificantIndex(s, i)
		if pj >= 0 {
			pv := s.At(pj)
			if pv.Kind == token.Punct && (pv.Value == "->" || pv.Value == "?->" || pv.Value == "::" || pv.Value == `\`) {
				continue
			}
			if pv.Kind == token.Keyword && strings.EqualFold(pv.Value, "function") {
				continue
			}
		}
		op := nextSignificantIndex(s, i)
		if op < 0 || s.At(op).Kind != token.Punct || s.At(op).Value != "(" {
			continue
		}
		cl := s.MatchForward(op)
		if cl < 0 {
			continue
		}
		start, neg := i, false
		if pj >= 0 && s.At(pj).Kind == token.Punct && s.At(pj).Value == "!" {
			neg, start = true, pj
		}
		sites = append(sites, site{start, op, cl, neg})
	}
	if len(sites) == 0 {
		return false
	}
	for _, st := range slices.Backward(sites) {
		a, b := st.argOpen+1, st.argClose-1
		for a <= b && s.At(a).Kind == token.Whitespace {
			a++
		}
		for b >= a && s.At(b).Kind == token.Whitespace {
			b--
		}
		var arg []token.Token
		for x := a; x <= b; x++ {
			arg = append(arg, s.At(x))
		}
		opTok := "==="
		if st.neg {
			opTok = "!=="
		}
		repl := []token.Token{
			{Kind: token.Ident, Value: "null"},
			{Kind: token.Whitespace, Value: " "},
			{Kind: token.Punct, Value: opTok},
			{Kind: token.Whitespace, Value: " "},
		}
		if argHasDepth0Operator(arg) {
			repl = append(repl, token.Token{Kind: token.Punct, Value: "("})
			repl = append(repl, arg...)
			repl = append(repl, token.Token{Kind: token.Punct, Value: ")"})
		} else {
			repl = append(repl, arg...)
		}
		s.ReplaceRange(st.start, st.argClose, repl)
	}
	return true
}

// argHasDepth0Operator reports whether the argument expression contains a
// top-level operator (anything but a postfix member/chain), meaning it must keep
// its parentheses to preserve precedence.
func argHasDepth0Operator(arg []token.Token) bool {
	depth := 0
	for _, t := range arg {
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				depth++
				continue
			case ")", "]", "}":
				depth--
				continue
			}
			if depth == 0 {
				switch t.Value {
				case "->", "?->", "::", `\`, "...":
				default:
					return true
				}
			}
		}
		if depth == 0 && t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "and", "or", "xor", "instanceof":
				return true
			}
		}
	}
	return false
}
