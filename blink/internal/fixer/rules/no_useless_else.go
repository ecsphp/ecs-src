package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoUselessElseFixer.php
//
// NoUselessElse removes an empty "else", or one whose preceding branches all
// exit. Like PHP-CS-Fixer, only the "else", "{" and "}" tokens go; later fixers
// fix the indentation of the body.
type NoUselessElse struct{}

func (NoUselessElse) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\NoUselessElseFixer`
}

func (NoUselessElse) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoUselessElseFixer.php"
}

func (NoUselessElse) Fix(s *tokens.Stream) bool {
	changed := false
	for index := 0; index < s.Len(); index++ {
		if !kwIs(s.At(index), "else") {
			continue
		}
		// "else if" and the alternative syntax "else:"
		next := sigNext(s, index)
		if next < 0 || isPunctVal(s, next, ":") || kwIs(s.At(next), "if") {
			continue
		}
		if nueFixEmptyElse(s, index) {
			changed = true
			continue
		}
		if nseIsSuperfluousElse(s, index) && !nueContainsNamedSymbolDeclaration(s, index) {
			nueClearElse(s, index)
			changed = true
		}
	}
	if changed {
		compactEmptyTokens(s)
	}
	return changed
}

// nueFixEmptyElse clears an "else" whose body is empty ("{}", "{ /**/ }" or
// "else;") and reports whether it did.
func nueFixEmptyElse(s *tokens.Stream, index int) bool {
	next := sigNext(s, index)
	if isPunctVal(s, next, "{") {
		closeIdx := s.MatchForward(next)
		if closeIdx < 0 {
			return false
		}
		if closeIdx-next == 1 || sigNext(s, next) == closeIdx {
			nueClearElse(s, index)
			return true
		}
		return false
	}
	// short "else"
	end := -1
	for j := index + 1; j < s.Len(); j++ {
		if isPunctVal(s, j, ";") || s.At(j).Kind == token.CloseTag {
			end = j
			break
		}
	}
	if next == end {
		nueClearElse(s, index)
		return true
	}
	return false
}

// nueClearElse mirrors clearElse: the "else" goes, and so do the braces of its
// block, each merging the whitespace around it.
func nueClearElse(s *tokens.Stream, index int) {
	clearTokenAndMergeWS(s, index)
	next := sigNext(s, index)
	if !isPunctVal(s, next, "{") {
		return
	}
	if closeIdx := s.MatchForward(next); closeIdx >= 0 {
		clearTokenAndMergeWS(s, closeIdx)
	}
	clearTokenAndMergeWS(s, next)
}

// nueContainsNamedSymbolDeclaration reports whether the "else" block declares
// a named function or a non-anonymous class-like.
func nueContainsNamedSymbolDeclaration(s *tokens.Stream, index int) bool {
	next := sigNext(s, index)
	if !isPunctVal(s, next, "{") {
		return false // a short "else" cannot declare a symbol
	}
	closeIdx := s.MatchForward(next)
	for i := next + 1; i < closeIdx; i++ {
		if kwIs(s.At(i), "function") && !taIsNamedArgument(s, i) && !taIsLambda(s, i) {
			return true
		}
		if taIsClassy(s, i) && !taIsAnonymousClass(s, i) {
			return true
		}
	}
	return false
}
