package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitMockShortWillReturnFixer.php
//
// PhpUnitMockShortWillReturn replaces `->will($this->returnValue(...))` and its
// returnArgument/returnCallback/returnSelf/returnValueMap siblings by the short
// `->willReturn(...)` form, inside PHPUnit test classes. Risky upstream: it
// assumes the methods are PHPUnit's own.
type PhpUnitMockShortWillReturn struct{}

// phpUnitMockShortWillReturnMap is upstream's RETURN_METHODS_MAP.
var phpUnitMockShortWillReturnMap = map[string]string{
	"returnargument": "willReturnArgument",
	"returncallback": "willReturnCallback",
	"returnself":     "willReturnSelf",
	"returnvalue":    "willReturn",
	"returnvaluemap": "willReturnMap",
}

func (PhpUnitMockShortWillReturn) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitMockShortWillReturnFixer`
}

func (PhpUnitMockShortWillReturn) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitMockShortWillReturnFixer.php"
}

func (PhpUnitMockShortWillReturn) Fix(s *tokens.Stream) bool {
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		for index := cls[0]; index < cls[1]; index++ {
			if !puIsObjectOperator(s, index) {
				continue
			}
			willIdx := sigNext(s, index)
			if willIdx < 0 || s.At(willIdx).Kind != token.Ident || !strings.EqualFold(s.At(willIdx).Value, "will") {
				continue
			}
			willOpen := sigNext(s, willIdx)
			if !isPunctVal(s, willOpen, "(") {
				continue
			}
			ref := sigNext(s, willOpen)
			op := sigNext(s, ref)
			fn := sigNext(s, op)
			if ref < 0 || op < 0 || fn < 0 || !puIsTheSameClassCall(s, fn) {
				continue
			}
			short, ok := phpUnitMockShortWillReturnMap[strings.ToLower(s.At(fn).Value)]
			if !ok {
				continue
			}
			open := sigNext(s, fn)
			// a first-class callable `$this->returnValue(...)`
			if ell := sigNext(s, open); isPunctVal(s, ell, "...") && isPunctVal(s, sigNext(s, ell), ")") {
				continue
			}
			closing := s.MatchForward(open)
			if closing < 0 {
				continue
			}
			s.Set(willIdx, token.Token{Kind: token.Ident, Value: short, Pos: s.At(willIdx).Pos})
			clearTokenAndMergeWS(s, ref)
			clearTokenAndMergeWS(s, op)
			clearTokenAndMergeWS(s, fn)
			clearTokenAndMergeWS(s, open)
			clearTokenAndMergeWS(s, closing)
			if comma := sigNext(s, closing); isPunctVal(s, comma, ",") {
				clearTokenAndMergeWS(s, comma)
			}
			changed = true
		}
	}
	if changed {
		s.ClearEmptyTokens()
	}
	return changed
}
