package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/StandaloneLinePlainConstructorParamFixer.php
//
// StandaloneLinePlainConstructorParam puts each parameter of a constructor
// without promoted properties on its own line, but only when it has 4 or more
// parameters. Promoted constructors are left to StandaloneLinePromotedProperty.
type StandaloneLinePlainConstructorParam struct{}

// Short parameter lists read fine on a single line.
const plainConstructorMinParamCount = 4

func (StandaloneLinePlainConstructorParam) Name() string {
	return `Symplify\CodingStandard\Fixer\Spacing\StandaloneLinePlainConstructorParamFixer`
}

func (StandaloneLinePlainConstructorParam) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/StandaloneLinePlainConstructorParamFixer.php"
}

func (StandaloneLinePlainConstructorParam) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Ident || !strings.EqualFold(s.At(i).Value, "__construct") {
			continue
		}
		if p := sigPrev(s, i); p < 0 || s.At(p).Kind != token.Keyword || strings.ToLower(s.At(p).Value) != "function" {
			continue
		}
		open := sigNext(s, i)
		if open < 0 || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || sigNext(s, open) == closeIdx {
			continue // no parameters
		}
		if countParams(s, open, closeIdx) < plainConstructorMinParamCount {
			continue
		}
		if hasPromotedParam(s, open, closeIdx) {
			continue // promoted constructors are handled by StandaloneLinePromotedProperty
		}
		if reflowParen(s, open, closeIdx, true) {
			changed = true
		}
	}
	return changed
}

// countParams counts top-level parameters (variables at nesting depth 0) in the paren.
func countParams(s *tokens.Stream, open, closeIdx int) int {
	count := 0
	depth := 0
	for j := open + 1; j < closeIdx; j++ {
		t := s.At(j)
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
			continue
		}
		if depth == 0 && t.Kind == token.Variable {
			count++
		}
	}
	return count
}
