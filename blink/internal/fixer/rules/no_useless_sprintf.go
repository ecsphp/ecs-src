package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NoUselessSprintfFixer.php
//
// NoUselessSprintf unwraps a global sprintf() call that has only its format
// argument: "sprintf('bar')" becomes "'bar'". A spread argument
// ("sprintf(...$args)") is kept. Risky: sprintf may be overridden.
type NoUselessSprintf struct{}

func (NoUselessSprintf) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\NoUselessSprintfFixer`
}

func (NoUselessSprintf) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/NoUselessSprintfFixer.php"
}

// IsCandidate rules out a file without the trigger: only a "sprintf" call is unwrapped.
func (NoUselessSprintf) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasWord("sprintf")
}

func (NoUselessSprintf) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 1; index > 0; index-- {
		t := s.At(index)
		if t.Kind != token.Ident || strings.ToLower(t.Value) != "sprintf" {
			continue
		}
		if !aliasIsGlobalFunctionCall(s, index) {
			continue
		}
		open := nextMeaningfulIndex(s, index) // the "(" (checked by aliasIsGlobalFunctionCall)
		if isPunctVal(s, nextMeaningfulIndex(s, open), "...") {
			continue
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
		changed = true
	}
	return changed
}
