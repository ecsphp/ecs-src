package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Symplify: https://github.com/symplify/coding-standard/blob/main/src/Fixer/Spacing/StandaloneLineConstructorParamFixer.php
//
// StandaloneLineConstructorParam puts each parameter of every constructor on
// its own line, with "(" and ")" on their own lines too, whatever the
// parameter count.
type StandaloneLineConstructorParam struct{}

func (StandaloneLineConstructorParam) Name() string {
	return `Symplify\CodingStandard\Fixer\Spacing\StandaloneLineConstructorParamFixer`
}

func (StandaloneLineConstructorParam) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Spacing/StandaloneLineConstructorParamFixer.php"
}

func (StandaloneLineConstructorParam) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		if s.At(i).Kind != token.Keyword || !strings.EqualFold(s.At(i).Value, "function") {
			continue
		}
		// MethodNameResolver compares the name case-sensitively
		if n := sigNext(s, i); n < 0 || s.At(n).Value != "__construct" {
			continue
		}
		open := nextParenOrSemicolon(s, i)
		if open < 0 {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || sigNext(s, open) == closeIdx {
			continue
		}
		if (LineLength{}).breakItems(s, open, closeIdx) {
			changed = true
		}
	}
	return changed
}

// nextParenOrSemicolon returns the first "(" after i, or -1 when a ";" comes
// first, as Symplify's BlockFinder looks for the parameter list.
func nextParenOrSemicolon(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		if s.At(j).Kind != token.Punct {
			continue
		}
		switch s.At(j).Value {
		case "(":
			return j
		case ";":
			return -1
		}
	}
	return -1
}
