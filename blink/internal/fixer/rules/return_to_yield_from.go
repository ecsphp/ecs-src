package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/ReturnToYieldFromFixer.php
//
// ReturnToYieldFrom turns "return [...]" into "yield from [...]" when the
// enclosing function declares the "iterable" return type and the returned array
// is its whole body.
type ReturnToYieldFrom struct{}

func (ReturnToYieldFrom) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\ReturnToYieldFromFixer`
}

func (ReturnToYieldFrom) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/ReturnToYieldFromFixer.php"
}

func (ReturnToYieldFrom) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if !kwIs(s.At(i), "return") {
			continue
		}
		if !rtyfShouldBeFixed(s, i) {
			continue
		}
		s.Set(i, token.Token{Kind: token.Keyword, Value: "yield from"})
		changed = true
	}
	return changed
}

func rtyfShouldBeFixed(s *tokens.Stream, returnIndex int) bool {
	arrayStart := aliasNextMeaningful(s, returnIndex)
	if arrayStart < 0 {
		return false
	}

	var arrayEnd int
	switch {
	case isPunctVal(s, arrayStart, "["):
		arrayEnd = s.MatchForward(arrayStart)
	case kwIs(s.At(arrayStart), "array"):
		open := nextPunctOfKind(s, arrayStart, "(")
		if open < 0 {
			return false
		}
		arrayEnd = s.MatchForward(open)
	default:
		return false
	}
	if arrayEnd < 0 {
		return false
	}

	functionEnd := arrayEnd
	for {
		functionEnd = aliasNextMeaningful(s, functionEnd)
		if functionEnd < 0 || !isPunctVal(s, functionEnd, ";") {
			break
		}
	}
	if functionEnd < 0 || !isPunctVal(s, functionEnd, "}") {
		return false
	}

	functionStart := s.MatchBackward(functionEnd)
	if functionStart < 0 {
		return false
	}

	returnTypeIndex := aliasPrevMeaningful(s, functionStart)
	if returnTypeIndex < 0 || s.At(returnTypeIndex).Kind != token.Ident {
		return false
	}
	if strings.ToLower(s.At(returnTypeIndex).Value) != "iterable" {
		return false
	}

	beforeReturnType := aliasPrevMeaningful(s, returnTypeIndex)
	return isPunctVal(s, beforeReturnType, ":") && isReturnTypeColon(s, beforeReturnType)
}
