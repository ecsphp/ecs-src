package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/symplify/coding-standard/blob/main/src/Fixer/Spacing/StandaloneLinePromotedPropertyFixer.php
//
// StandaloneLinePromotedProperty is a file-level fixer: when any constructor in
// the file declares a promoted property, it puts each parameter of every
// constructor in the file on its own line ("(" and ")" on their own lines),
// promoted or plain. A file with no promoted constructor is left untouched (plain
// constructors with 4+ params are then handled by StandaloneLinePlainConstructorParam).
type StandaloneLinePromotedProperty struct{}

func (StandaloneLinePromotedProperty) Name() string {
	return `Symplify\CodingStandard\Fixer\Spacing\StandaloneLinePromotedPropertyFixer`
}

func (StandaloneLinePromotedProperty) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Spacing/StandaloneLinePromotedPropertyFixer.php"
}

func (StandaloneLinePromotedProperty) Fix(s *tokens.Stream) bool {
	if !fileHasPromotedConstructor(s) {
		return false // the fixer only runs when a promoted constructor is present
	}
	changed := false
	for i := 0; i < s.Len(); i++ {
		open := constructorParamOpen(s, i)
		if open < 0 {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || sigNext(s, open) == closeIdx {
			continue // no parameters
		}
		if reflowParen(s, open, closeIdx, false) {
			changed = true
		}
	}
	return changed
}

// fileHasPromotedConstructor reports whether any constructor in the file declares
// a promoted property. Symplify's StandaloneLinePromotedPropertyFixer is a file-
// level candidate: once one promoted constructor exists it breaks the params of
// every constructor in the file onto standalone lines, promoted or plain.
func fileHasPromotedConstructor(s *tokens.Stream) bool {
	for i := 0; i < s.Len(); i++ {
		open := constructorParamOpen(s, i)
		if open < 0 {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 {
			continue
		}
		if hasPromotedParam(s, open, closeIdx) {
			return true
		}
	}
	return false
}

// constructorParamOpen returns the index of the "(" opening the parameter list of
// a "function __construct(" declaration at i, or -1 when i is not such a name.
func constructorParamOpen(s *tokens.Stream, i int) int {
	if s.At(i).Kind != token.Ident || !strings.EqualFold(s.At(i).Value, "__construct") {
		return -1
	}
	if p := sigPrev(s, i); p < 0 || s.At(p).Kind != token.Keyword || strings.ToLower(s.At(p).Value) != "function" {
		return -1
	}
	open := sigNext(s, i)
	if open < 0 || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
		return -1
	}
	return open
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
