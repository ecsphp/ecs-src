package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/IsNullFixer.php
//
// IsNull rewrites is_null($x) to "null === $x" and "!is_null($x)" to
// "null !== $x". Only global calls are rewritten (FunctionsAnalyzer::
// isGlobalFunctionCall), and not a call without argument, with an unpacked
// argument or with a named argument other than "value". The argument keeps its
// parentheses when it contains "?", "=" or "??", and the result is wrapped in
// parentheses after a cast or next to "==", "!=", "<>", "===" or "!==".
type IsNull struct{}

func (IsNull) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\IsNullFixer`
}

func (IsNull) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/IsNullFixer.php"
}

// IsCandidate rules out a file without the trigger: only an "is_null" call is replaced.
func (IsNull) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasWord("is_null")
}

func (IsNull) Fix(s *tokens.Stream) bool {
	fs := newFuncScan(sharedSymScan(s))
	var calls []int
	for i := range s.Len() {
		if t := s.At(i); t.Kind == token.Ident && strings.EqualFold(t.Value, "is_null") && fs.globalFunction(i, true) {
			calls = append(calls, i)
		}
	}
	// Calls are fixed in order, each one seeing the edits of the calls before
	// it (an outer call is fixed before the calls in its argument). Edits clear
	// tokens instead of removing them, so only the inserted tokens move the
	// calls still to fix.
	changed := false
	for k, call := range calls {
		closeParen, inserted, ok := fixIsNullCall(s, call)
		if !ok {
			continue
		}
		changed = true
		for r := k + 1; r < len(calls); r++ {
			if inserted.closing && calls[r] > closeParen {
				calls[r]++
			}
			calls[r] += inserted.opening
		}
	}
	return changed
}

// isNullInsertion counts the tokens fixIsNullCall inserted: opening at the
// call's name (the name itself becomes several tokens) and closing (a ")")
// after the call's closing parenthesis.
type isNullInsertion struct {
	opening int
	closing bool
}

// fixIsNullCall ports the body of IsNullFixer::applyFix's loop for the global
// call at name. It returns the index of the call's closing parenthesis before
// the tokens were inserted, and false when the call is left as is.
func fixIsNullCall(s *tokens.Stream, name int) (int, isNullInsertion, bool) {
	open := aliasNextMeaningful(s, name)
	next := aliasNextMeaningful(s, open)
	if next < 0 || isPunctVal(s, next, ")") || isPunctVal(s, next, "...") {
		return 0, isNullInsertion{}, false
	}
	closeParen := s.MatchForward(open)
	if closeParen < 0 {
		return 0, isNullInsertion{}, false
	}

	// is_null() takes a single parameter, so only the first token inside the
	// parentheses can carry a name
	if colon := aliasNextMeaningful(s, next); isPunctVal(s, colon, ":") && (s.Kind(next) == token.Ident || s.Kind(next) == token.Keyword) {
		if s.At(next).Value != "value" {
			return 0, isNullInsertion{}, false
		}
		clearTokenAndMergeWS(s, colon)
		clearTokenAndMergeWS(s, next)
		removeWhitespaceSafely(s, open, 1)
	}

	prev := aliasPrevMeaningful(s, name)
	if isPunctVal(s, prev, `\`) {
		removeWhitespaceSafely(s, prev, 1)
		clearTokenAt(s, prev)
		prev = aliasPrevMeaningful(s, prev)
	}
	inverted := isPunctVal(s, prev, "!")
	if inverted {
		removeWhitespaceSafely(s, prev, 1)
		clearTokenAt(s, prev)
	}

	keepParentheses := false
	for i := open; i <= closeParen; i++ {
		switch s.At(i).Value {
		case "?", "?:", "=", "??":
			keepParentheses = true
		}
	}

	left := aliasPrevMeaningful(s, name)
	wrap := isNullParentOperation(s, left) || isNullParentOperation(s, aliasNextMeaningful(s, closeParen)) ||
		(isPunctVal(s, left, ")") && isCastCloseParen(s, left))

	if comma := aliasPrevMeaningful(s, closeParen); isPunctVal(s, comma, ",") {
		clearTokenAndMergeWS(s, comma)
	}

	if !keepParentheses {
		removeWhitespaceSafely(s, closeParen, -1)
		clearTokenAt(s, closeParen)
		removeWhitespaceSafely(s, open, -1)
		removeWhitespaceSafely(s, open, 1)
		clearTokenAt(s, open)
	}

	operator := "==="
	if inverted {
		operator = "!=="
	}
	replacement := []token.Token{
		{Kind: token.Ident, Value: "null"},
		{Kind: token.Whitespace, Value: " "},
		{Kind: token.Punct, Value: operator},
		{Kind: token.Whitespace, Value: " "},
	}
	if wrap {
		replacement = append([]token.Token{{Kind: token.Punct, Value: "("}}, replacement...)
		s.InsertAt(closeParen+1, token.Token{Kind: token.Punct, Value: ")"})
	}
	s.ReplaceRange(name, name, replacement)

	return closeParen, isNullInsertion{opening: len(replacement) - 1, closing: wrap}, true
}

// isNullParentOperation reports whether the token at i is one of the
// comparisons that make IsNullFixer wrap its result in parentheses.
func isNullParentOperation(s *tokens.Stream, i int) bool {
	if i < 0 || s.Kind(i) != token.Punct {
		return false
	}
	switch s.At(i).Value {
	case "==", "!=", "<>", "===", "!==":
		return true
	}
	return false
}
