package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/StaticPrivateMethodFixer.php
//
// StaticPrivateMethod converts private methods that never touch the instance to
// static (risky), and rewrites their "$this->method()" callers to "self::". It
// loops to a fixpoint so chains of mutually calling methods resolve.
type StaticPrivateMethod struct{}

func (StaticPrivateMethod) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\StaticPrivateMethodFixer`
}

func (StaticPrivateMethod) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/StaticPrivateMethodFixer.php"
}

var spmMagicMethods = map[string]bool{
	"__clone": true, "__construct": true, "__destruct": true, "__wakeup": true,
}

type spmMethod struct {
	funcIndex int
	open      int
	close     int
}

func (StaticPrivateMethod) Fix(s *tokens.Stream) bool {
	changed := false
	for {
		anythingChanged := false
		for index := s.Len() - 3; index > 0; index-- {
			if !taIsClassKw(s, index) {
				continue
			}
			classOpen := nextPunctOfKind(s, index, "{")
			if classOpen < 0 {
				continue
			}
			classClose := s.MatchForward(classOpen)
			if classClose < 0 {
				continue
			}
			if spmFixClass(s, classOpen, classClose) {
				anythingChanged = true
			}
		}
		if !anythingChanged {
			break
		}
		changed = true
	}
	return changed
}

func spmFixClass(s *tokens.Stream, classOpen, classClose int) bool {
	fixedMethods := map[string]bool{}
	// snapshot before inserting; methods are visited high index to low, so each
	// insert of "static " shifts only tokens above an already-visited method
	for _, m := range spmClassMethods(s, classOpen, classClose) {
		if spmSkipMethod(s, m.funcIndex, m.open, m.close) {
			continue
		}
		nameIndex := sigNext(s, m.funcIndex)
		fixedMethods[s.At(nameIndex).Value] = true
		s.InsertSliceAt(m.funcIndex, []token.Token{
			{Kind: token.Keyword, Value: "static"},
			{Kind: token.Whitespace, Value: " "},
		})
	}

	if len(fixedMethods) == 0 {
		return false
	}

	classClose = s.MatchForward(classOpen)
	for _, m := range spmClassMethods(s, classOpen, classClose) {
		spmFixReferences(s, m.open, m.close, fixedMethods)
	}
	return true
}

// spmSkipMethod mirrors StaticPrivateMethodFixer::skipMethod.
func spmSkipMethod(s *tokens.Stream, funcIndex, methodOpen, methodClose int) bool {
	nameIndex := sigNext(s, funcIndex)
	methodName := strings.ToLower(s.At(nameIndex).Value)
	if spmMagicMethods[methodName] {
		return true
	}

	prev := sigPrev(s, funcIndex)
	if kwIs(s.At(prev), "final") {
		prev = sigPrev(s, prev)
	}
	if prev < 0 || !kwIs(s.At(prev), "private") {
		return true
	}
	if prePrev := sigPrev(s, prev); prePrev >= 0 && kwIs(s.At(prePrev), "static") {
		return true
	}

	for index := methodOpen + 1; index < methodClose-1; index++ {
		if taIsClassKw(s, index) && taIsAnonymousClass(s, index) {
			anonOpen := nextPunctOfKind(s, index, "{")
			if end := s.MatchForward(anonOpen); end >= 0 {
				index = end
			}
			continue
		}
		if kwIs(s.At(index), "function") {
			return true
		}
		if t := s.At(index); t.Kind == token.Variable && t.Value == "$this" {
			operatorIndex := sigNext(s, index)
			nameIdx := sigNext(s, operatorIndex)
			argsBrace := sigNext(s, nameIdx)
			if !isPunctVal(s, operatorIndex, "->") ||
				methodName != s.At(nameIdx).Value ||
				!isPunctVal(s, argsBrace, "(") {
				return true
			}
		}
		if t := s.At(index); t.Kind == token.Ident && t.Value == "debug_backtrace" {
			return true
		}
	}
	return false
}

// spmFixReferences rewrites "$this->method()" to "self::method()" for methods in
// fixedMethods, recursing into non-static closures.
func spmFixReferences(s *tokens.Stream, methodOpen, methodClose int, fixedMethods map[string]bool) {
	for index := methodOpen + 1; index < methodClose-1; index++ {
		if kwIs(s.At(index), "function") {
			prevIndex := sigPrev(s, index)
			closureStart := nextPunctOfKind(s, index, "{")
			if closureStart < 0 {
				continue
			}
			closureEnd := s.MatchForward(closureStart)
			if closureEnd < 0 {
				continue
			}
			if prevIndex < 0 || !kwIs(s.At(prevIndex), "static") {
				spmFixReferences(s, closureStart, closureEnd, fixedMethods)
			}
			index = closureEnd
			continue
		}
		if taIsClassKw(s, index) && taIsAnonymousClass(s, index) {
			anonOpen := nextPunctOfKind(s, index, "{")
			if end := s.MatchForward(anonOpen); end >= 0 {
				index = end
			}
			continue
		}
		if t := s.At(index); t.Kind != token.Variable || t.Value != "$this" {
			continue
		}
		operatorIndex := sigNext(s, index)
		if !isPunctVal(s, operatorIndex, "->") {
			continue
		}
		nameIdx := sigNext(s, operatorIndex)
		argsBrace := sigNext(s, nameIdx)
		if !isPunctVal(s, argsBrace, "(") {
			continue
		}
		if !fixedMethods[s.At(nameIdx).Value] {
			continue
		}
		s.Set(index, token.Token{Kind: token.Ident, Value: "self"})
		s.Set(operatorIndex, token.Token{Kind: token.Punct, Value: "::"})
	}
}

// spmClassMethods mirrors StaticPrivateMethodFixer::getClassMethods: non-abstract
// methods of the class body, in reverse source order.
func spmClassMethods(s *tokens.Stream, classOpen, classClose int) []spmMethod {
	var out []spmMethod
	for index := classClose - 1; index > classOpen+1; index-- {
		if isPunctVal(s, index, "}") {
			if start := s.MatchBackward(index); start > classOpen {
				index = start
			}
			continue
		}
		if !kwIs(s.At(index), "function") {
			continue
		}
		if spmMethodIsAbstract(s, index) {
			continue
		}
		methodOpen := nextPunctOfKind(s, index, "{")
		if methodOpen < 0 {
			continue
		}
		methodClose := s.MatchForward(methodOpen)
		if methodClose < 0 {
			continue
		}
		out = append(out, spmMethod{funcIndex: index, open: methodOpen, close: methodClose})
	}
	return out
}

// spmMethodIsAbstract mirrors TokensAnalyzer::getMethodAttributes()['abstract'].
func spmMethodIsAbstract(s *tokens.Stream, funcIndex int) bool {
	for j := sigPrev(s, funcIndex); j >= 0 && taIsModifier(s, j); j = sigPrev(s, j) {
		if kwIs(s.At(j), "abstract") {
			return true
		}
	}
	return false
}
