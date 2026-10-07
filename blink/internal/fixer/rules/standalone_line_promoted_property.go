package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/symplify/coding-standard/blob/main/src/Fixer/Spacing/StandaloneLinePromotedPropertyFixer.php
//
// StandaloneLinePromotedProperty puts each constructor parameter on its own line:
// a "function __construct(...)" with at least one parameter becomes fully
// multiline, "(" and ")" on their own lines. Matches symplify's constructor
// param newliners (promoted and plain), which together break every __construct.
type StandaloneLinePromotedProperty struct{}

func (StandaloneLinePromotedProperty) Name() string {
	return `Symplify\CodingStandard\Fixer\Spacing\StandaloneLinePromotedPropertyFixer`
}

func (StandaloneLinePromotedProperty) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Spacing/StandaloneLinePromotedPropertyFixer.php"
}

func (StandaloneLinePromotedProperty) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Ident || !strings.EqualFold(s.At(i).Value, "__construct") {
			continue
		}
		// must be a declaration: "function __construct("
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
		if !hasPromotedParam(s, open, closeIdx) {
			continue // only constructor property promotion is split out
		}
		if reflowParen(s, open, closeIdx, false) {
			changed = true
		}
	}
	return changed
}

// hasPromotedParam reports whether the parameter list has a promoted property -
// a parameter carrying a public/protected/private/readonly modifier.
func hasPromotedParam(s *tokens.Stream, open, closeIdx int) bool {
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
		if depth == 0 && t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "public", "protected", "private", "readonly":
				return true
			}
		}
	}
	return false
}
