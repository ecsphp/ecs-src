package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/NoBlankLineBetweenImportsFixer.php
//
// NoBlankLineBetweenImports removes blank lines between consecutive import "use"
// statements. The blank line after the namespace (before the first import) is
// left untouched.
type NoBlankLineBetweenImports struct{}

func (NoBlankLineBetweenImports) Name() string {
	return `Symplify\CodingStandard\Fixer\Spacing\NoBlankLineBetweenImportsFixer`
}

func (NoBlankLineBetweenImports) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/NoBlankLineBetweenImportsFixer.php"
}

func (NoBlankLineBetweenImports) Fix(s *tokens.Stream) bool {
	// collect import "use" indexes (skip in-class trait use and closure use)
	var uses []int
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Keyword || strings.ToLower(s.At(i).Value) != "use" {
			continue
		}
		if inClassLikeBody(s, i) {
			continue
		}
		if j := skipWhitespace(s, i+1); j < s.Len() && s.At(j).Kind == token.Punct && s.At(j).Value == "(" {
			continue
		}
		uses = append(uses, i)
	}

	changed := false
	// start at the 2nd use, so the blank line after the namespace stays
	for k := 1; k < len(uses); k++ {
		useIndex := uses[k]
		prev := sigPrev(s, useIndex)
		if prev < 0 || s.At(prev).Kind != token.Punct || s.At(prev).Value != ";" {
			continue
		}
		ws := useIndex - 1
		if ws < 0 || s.At(ws).Kind != token.Whitespace {
			continue
		}
		if strings.Count(s.At(ws).Value, "\n") < 2 {
			continue
		}
		s.SetValue(ws, "\n")
		changed = true
	}
	return changed
}
