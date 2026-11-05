package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitDataProviderReturnTypeFixer.php
//
// PhpUnitDataProviderReturnType forces the return type of a PHPUnit data
// provider to `iterable`. Risky: relies on the provider's signature.
type PhpUnitDataProviderReturnType struct{}

func (PhpUnitDataProviderReturnType) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitDataProviderReturnTypeFixer`
}

func (PhpUnitDataProviderReturnType) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitDataProviderReturnTypeFixer.php"
}

func (PhpUnitDataProviderReturnType) Fix(s *tokens.Stream) bool {
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		if dataProviderReturnTypeClass(s, cls[0], cls[1]) {
			changed = true
		}
	}
	return changed
}

func dataProviderReturnTypeClass(s *tokens.Stream, start, end int) bool {
	providers := puDataProviders(s, start, end)
	changed := false
	// reverse order so an insert never shifts a provider not yet visited
	for i := len(providers) - 1; i >= 0; i-- {
		if dataProviderReturnTypeOne(s, providers[i].nameIndex) {
			changed = true
		}
	}
	return changed
}

func dataProviderReturnTypeOne(s *tokens.Stream, nameIndex int) bool {
	open := nextPunctOfKind(s, nameIndex, "(")
	if open < 0 {
		return false
	}
	argsEnd := s.MatchForward(open)
	if argsEnd < 0 {
		return false
	}
	colon := sigNext(s, argsEnd)
	if !isPunctVal(s, colon, ":") {
		s.InsertSliceAt(argsEnd+1, []token.Token{
			{Kind: token.Punct, Value: ":"},
			{Kind: token.Whitespace, Value: " "},
			{Kind: token.Ident, Value: "iterable"},
		})
		return true
	}

	bodyOrSemi := nextPunctOfKind(s, argsEnd, "{", ";")
	if bodyOrSemi < 0 {
		return false
	}
	typeStart := sigNext(s, colon)
	typeEnd := sigPrev(s, bodyOrSemi)
	if typeStart < 0 || typeEnd < typeStart {
		return false
	}

	var b strings.Builder
	for j := typeStart; j <= typeEnd; j++ {
		if k := s.At(j).Kind; k == token.Whitespace || k == token.Comment || k == token.DocComment {
			continue
		}
		b.WriteString(s.At(j).Value)
	}
	if b.String() == "iterable" {
		return false
	}
	s.ReplaceRange(typeStart, typeEnd, []token.Token{{Kind: token.Ident, Value: "iterable"}})
	return true
}
