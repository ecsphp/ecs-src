package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/FinalPublicMethodForAbstractClassFixer.php
//
// FinalPublicMethodForAbstractClass marks every public method of an abstract
// class final (risky), except magic methods and already abstract/final ones.
type FinalPublicMethodForAbstractClass struct{}

func (FinalPublicMethodForAbstractClass) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\FinalPublicMethodForAbstractClassFixer`
}

func (FinalPublicMethodForAbstractClass) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/FinalPublicMethodForAbstractClassFixer.php"
}

// fpmMagicMethods are the magic methods left non-final.
var fpmMagicMethods = map[string]bool{
	"__construct": true, "__destruct": true, "__call": true, "__callstatic": true,
	"__get": true, "__set": true, "__isset": true, "__unset": true, "__sleep": true,
	"__wakeup": true, "__tostring": true, "__invoke": true, "__set_state": true,
	"__clone": true, "__debuginfo": true,
}

func (FinalPublicMethodForAbstractClass) Fix(s *tokens.Stream) bool {
	changed := false
	// collect T_ABSTRACT indices, then process in reverse (insertions shift only
	// indices to the right)
	var abstracts []int
	for i := 0; i < s.Len(); i++ {
		if kwIs(s.At(i), "abstract") {
			abstracts = append(abstracts, i)
		}
	}
	for _, abstractIndex := range slices.Backward(abstracts) {
		classIndex := fpmNextClassOrFunction(s, abstractIndex)
		if classIndex < 0 || !taIsClassKw(s, classIndex) {
			continue
		}
		classOpen := nextPunctOfKind(s, classIndex, "{")
		if classOpen < 0 {
			continue
		}
		classClose := s.MatchForward(classOpen)
		if classClose < 0 {
			continue
		}
		if fpmFixClass(s, classOpen, classClose) {
			changed = true
		}
	}
	return changed
}

// fpmNextClassOrFunction returns the index of the first "class" or "function"
// keyword after index, or -1.
func fpmNextClassOrFunction(s *tokens.Stream, index int) int {
	for i := index + 1; i < s.Len(); i++ {
		if taIsClassKw(s, i) || kwIs(s.At(i), "function") {
			return i
		}
	}
	return -1
}

func fpmFixClass(s *tokens.Stream, classOpen, classClose int) bool {
	changed := false
	for index := classClose - 1; index > classOpen; index-- {
		if isPunctVal(s, index, "}") {
			if start := s.MatchBackward(index); start > classOpen {
				index = start
			}
			continue
		}
		if !kwIs(s.At(index), "public") {
			continue
		}
		nextIndex := sigNext(s, index)
		if kwIs(s.At(nextIndex), "static") {
			nextIndex = sigNext(s, nextIndex)
		}
		if nextIndex < 0 || !kwIs(s.At(nextIndex), "function") {
			continue // use, attribute, constant, etc.
		}
		nameIndex := sigNext(s, nextIndex)
		if nameIndex >= 0 && fpmMagicMethods[strings.ToLower(s.At(nameIndex).Value)] {
			continue
		}
		prevIndex := sigPrev(s, index)
		if kwIs(s.At(prevIndex), "static") {
			index = prevIndex
			prevIndex = sigPrev(s, index)
		}
		if prevIndex >= 0 && (kwIs(s.At(prevIndex), "abstract") || kwIs(s.At(prevIndex), "final")) {
			index = prevIndex
			continue
		}
		s.InsertSliceAt(index, []token.Token{
			{Kind: token.Keyword, Value: "final"},
			{Kind: token.Whitespace, Value: " "},
		})
		changed = true
	}
	return changed
}
