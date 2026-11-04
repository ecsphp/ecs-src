package rules

import (
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/PowToExponentiationFixer.php
//
// PowToExponentiation converts a global "pow($a, $b)" call to "$a ** $b" (risky:
// pow may be overridden). An operand that is more than a simple value (a
// variable, number, name, literal string, int/float cast, member or static
// access, bracketed or parenthesised group) is wrapped in parentheses.
type PowToExponentiation struct{}

func (PowToExponentiation) Name() string {
	return `PhpCsFixer\Fixer\Alias\PowToExponentiationFixer`
}

func (PowToExponentiation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/PowToExponentiationFixer.php"
}

func (PowToExponentiation) Fix(s *tokens.Stream) bool {
	// minimal candidate to fix is seven tokens: pow(x,y);
	if s.Len() <= 7 {
		return false
	}
	s.ClearEmptyTokens()

	type call struct{ fn, open, close int }
	var candidates []call
	end := s.Len() - 6
	for i := 1; i < end; i++ {
		fn, open, closeIdx, ok := aliasFind(s, "pow", i, end)
		if !ok {
			break
		}
		i = open
		candidates = append(candidates, call{fn, open, closeIdx})
	}

	changed := false
	numberOfTokensAdded := 0
	previousCloseParenthesisIndex := s.Len()
	for _, c := range slices.Backward(candidates) {
		if c.close < 0 {
			continue
		}
		// tokens added inside this candidate by the previous (nested) ones moved
		// its closing parenthesis
		if previousCloseParenthesisIndex < c.close {
			previousCloseParenthesisIndex = c.close
			c.close += numberOfTokensAdded
		} else {
			previousCloseParenthesisIndex = c.close
			numberOfTokensAdded = 0
		}

		arguments := aliasArguments(s, c.open, c.close)
		if len(arguments) != 2 {
			continue
		}
		spread := false
		for i := c.open; i < c.close; i++ {
			if aliasIsEllipsis(s, i) {
				spread = true
				break
			}
		}
		if spread {
			continue
		}
		numberOfTokensAdded += powFix(s, c.fn, c.open, c.close, arguments)
		changed = true
	}
	if changed {
		aliasRelex(s)
	}
	return changed
}

// powFix rewrites one call and returns the number of tokens it added.
func powFix(s *tokens.Stream, fn, open, closeIdx int, arguments [][2]int) int {
	// the argument separator after the first argument becomes "**"
	for j := arguments[0][1] + 1; j < s.Len(); j++ {
		if isPunctVal(s, j, ",") {
			s.Set(j, aliasPunct("**"))
			break
		}
	}

	clearTokenAt(s, closeIdx)
	if p := aliasPrevMeaningful(s, closeIdx); isPunctVal(s, p, ",") {
		clearTokenAt(s, p) // trailing comma
	}

	added := 0
	for _, a := range slices.Backward(arguments) {
		if powIsParenthesisNeeded(s, a[0], a[1]) {
			s.InsertAt(a[1]+1, aliasPunct(")"))
			s.InsertAt(a[0], aliasPunct("("))
			added += 2
		}
	}

	clearTokenAt(s, open)
	clearTokenAndMergeWS(s, fn)
	if p := aliasPrevMeaningful(s, fn); isPunctVal(s, p, `\`) {
		clearTokenAt(s, p)
	}
	return added
}

// powIsParenthesisNeeded ports PowToExponentiationFixer::isParenthesisNeeded:
// the argument holds a token outside the allowed simple-operand kinds.
func powIsParenthesisNeeded(s *tokens.Stream, start, end int) bool {
	for i := start; i <= end; i++ {
		if powIsAllowed(s, i) || s.At(i).Value == "" {
			continue
		}
		if closeIdx, typ, ok := aliasCastAt(s, i); ok {
			switch typ {
			case "int", "integer", "float", "double", "real":
				i = closeIdx // T_INT_CAST / T_DOUBLE_CAST are allowed
				continue
			}
			return true // other casts are not
		}
		if blockEnd := aliasBlockEnd(s, i); blockEnd >= 0 {
			i = blockEnd
			continue
		}
		if t := s.At(i); t.Kind == token.Variable && t.Value == "$" {
			i = aliasNextMeaningful(s, i)
			if i < 0 {
				return true
			}
			if isPunctVal(s, i, "{") {
				// CT::T_DYNAMIC_VAR_BRACE_OPEN "${...}"
				i = s.MatchForward(i)
				if i < 0 {
					return true
				}
				continue
			}
		}
		if isPunctVal(s, i, "+") && aliasPrevMeaningful(s, i) < start {
			continue // unary plus leading the argument
		}
		return true
	}
	return false
}

// powIsAllowed mirrors PowToExponentiationFixer::getAllowedKinds.
func powIsAllowed(s *tokens.Stream, i int) bool {
	t := s.At(i)
	switch t.Kind {
	case token.Number, token.Whitespace, token.Comment, token.DocComment:
		return true
	case token.Variable:
		return t.Value != "$"
	case token.Ident:
		// T_STRING, plus T_LINE among the magic constants
		return aliasIsTString(s, i) || strings.EqualFold(t.Value, "__LINE__")
	case token.String:
		return aliasIsConstantString(t)
	case token.Keyword:
		return aliasIsNamespaceOperator(s, i)
	case token.Punct:
		switch t.Value {
		case "++", "--", `\`, "::", "->", "?->":
			return true
		}
	}
	return false
}
