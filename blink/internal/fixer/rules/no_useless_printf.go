package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NoUselessPrintfFixer.php
//
// NoUselessPrintf rewrites a global printf() call that has only its format
// argument to "print": "printf('bar')" becomes "print 'bar'". A spread or
// first-class callable argument is kept. Risky: printf may be overridden.
type NoUselessPrintf struct{}

func (NoUselessPrintf) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\NoUselessPrintfFixer`
}

func (NoUselessPrintf) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NoUselessPrintfFixer.php"
}

func (NoUselessPrintf) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 1; index > 0; index-- {
		t := s.At(index)
		if t.Kind != token.Ident || strings.ToLower(t.Value) != "printf" {
			continue
		}
		if !aliasIsGlobalFunctionCall(s, index) {
			continue
		}
		open := nextMeaningfulIndex(s, index) // the "(" (checked by aliasIsGlobalFunctionCall)
		if n := nextMeaningfulIndex(s, open); isPunctVal(s, n, "...") {
			continue // spread or first-class callable "printf(...)"
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || len(aliasArguments(s, open, closeIdx)) != 1 {
			continue
		}
		clearTokenAndMergeWS(s, closeIdx)
		if p := prevMeaningfulIndex(s, closeIdx); isPunctVal(s, p, ",") {
			clearTokenAndMergeWS(s, p)
		}
		clearTokenAndMergeWS(s, open)
		clearTokenAndMergeWS(s, index)
		if p := prevMeaningfulIndex(s, index); isPunctVal(s, p, `\`) {
			clearTokenAndMergeWS(s, p)
		}
		s.InsertSliceAt(index, []token.Token{
			{Kind: token.Keyword, Value: "print"},
			{Kind: token.Whitespace, Value: " "},
		})
		changed = true
	}
	return changed
}
