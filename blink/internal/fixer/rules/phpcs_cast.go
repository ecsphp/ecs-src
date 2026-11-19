package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// phpCastTypes are the type words PHP's lexer turns "(" type ")" into a cast token for.
var phpCastTypes = map[string]bool{
	"int": true, "integer": true, "bool": true, "boolean": true,
	"float": true, "double": true, "real": true, "string": true, "binary": true,
	"array": true, "object": true, "unset": true, "void": true,
}

// phpCastAt matches a cast token the way PHP lexes one, regardless of context:
// "(" + spaces/tabs + type + spaces/tabs + ")". It returns the index of ")"
// and the type word's index.
func phpCastAt(s *tokens.Stream, open int) (closeIdx, typeIdx int, ok bool) {
	if s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
		return 0, 0, false
	}
	j := open + 1
	if j < s.Len() && isCastPadding(s.At(j)) {
		j++
	}
	if j >= s.Len() {
		return 0, 0, false
	}
	t := s.At(j)
	if (t.Kind != token.Ident && t.Kind != token.Keyword) || !phpCastTypes[strings.ToLower(t.Value)] {
		return 0, 0, false
	}
	typeIdx = j
	j++
	if j < s.Len() && isCastPadding(s.At(j)) {
		j++
	}
	if j >= s.Len() || s.At(j).Kind != token.Punct || s.At(j).Value != ")" {
		return 0, 0, false
	}
	return j, typeIdx, true
}
