package rules

import (
	"slices"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/InnerFunctionsSniff.php
//
// InnerFunctions reports a named function declared inside a function or
// closure body, unless a class, interface, trait or enum sits in between.
type InnerFunctions struct{}

func (InnerFunctions) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\InnerFunctionsSniff`
}

func (InnerFunctions) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/InnerFunctionsSniff.php"
}

func (InnerFunctions) Fix(*tokens.Stream) bool { return false }

// innerFunctionsScope is what owns a "{": a function or closure, an OO
// structure, or anything else.
type innerFunctionsScope int

const (
	innerFunctionsOther innerFunctionsScope = iota
	innerFunctionsFunction
	innerFunctionsOO
)

func (f InnerFunctions) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	var errs []fixer.CodingStandardError
	var lines []int
	owners := map[int]innerFunctionsScope{}
	var stack []innerFunctionsScope
	for i := range s.Len() {
		t := s.At(i)
		switch {
		case isPunctVal(s, i, "{"):
			stack = append(stack, owners[i])
		case isPunctVal(s, i, "}"):
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case kwIs(t, "function"):
			named, body := innerFunctionsDeclaration(s, i)
			if body >= 0 {
				owners[body] = innerFunctionsFunction
			}
			if named && innerFunctionsNested(stack) {
				if lines == nil {
					lines = phpcsTokenLines(s)
				}
				errs = append(errs, fixer.CodingStandardError{
					Line:    lines[i],
					Message: "The use of inner functions is forbidden",
					Checker: f.Name() + ".NotAllowed",
				})
			}
		case kwIs(t, "class") || kwIs(t, "interface") || kwIs(t, "trait") || kwIs(t, "enum"):
			if p := sigPrev(s, i); p >= 0 && (isPunctVal(s, p, "::") || isPunctVal(s, p, "->") || isPunctVal(s, p, "?->")) {
				continue
			}
			if body := innerFunctionsBody(s, i+1); body >= 0 {
				owners[body] = innerFunctionsOO
			}
		}
	}
	return errs
}

// innerFunctionsNested reports whether the innermost function or OO scope
// around a declaration is a function or closure.
func innerFunctionsNested(stack []innerFunctionsScope) bool {
	for _, scope := range slices.Backward(stack) {
		switch scope {
		case innerFunctionsFunction:
			return true
		case innerFunctionsOO:
			return false
		}
	}
	return false
}

// innerFunctionsDeclaration tells a named function (T_FUNCTION) from a closure
// for the "function" keyword at i, and finds the "{" of its body (-1 for none).
func innerFunctionsDeclaration(s *tokens.Stream, i int) (named bool, body int) {
	if p := sigPrev(s, i); p >= 0 && (isPunctVal(s, p, "::") || isPunctVal(s, p, "->") || isPunctVal(s, p, "?->")) {
		return false, -1
	}
	n := sigNext(s, i)
	if isPunctVal(s, n, "&") {
		n = sigNext(s, n)
	}
	if n < 0 {
		return false, -1
	}
	named = !isPunctVal(s, n, "(")
	if named {
		n = sigNext(s, n)
	}
	if !isPunctVal(s, n, "(") {
		// "use function foo;" imports a function and declares none
		return named, -1
	}
	return named, innerFunctionsBody(s, n)
}

// innerFunctionsBody finds the "{" opening the body that starts after i,
// skipping parenthesized lists; -1 when a ";" ends the declaration first.
func innerFunctionsBody(s *tokens.Stream, i int) int {
	for j := i; j < s.Len(); j++ {
		switch {
		case isPunctVal(s, j, "{"):
			return j
		case isPunctVal(s, j, ";"):
			return -1
		case isPunctVal(s, j, "("):
			if m := s.MatchForward(j); m > j {
				j = m
			}
		}
	}
	return -1
}
