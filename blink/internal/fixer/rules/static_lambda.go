package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/StaticLambdaFixer.php
//
// StaticLambda makes closures and arrow functions "static" when they cannot
// use $this (risky).
type StaticLambda struct{}

func (StaticLambda) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\StaticLambdaFixer`
}

func (StaticLambda) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/StaticLambdaFixer.php"
}

func (StaticLambda) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 4; index > 0; index-- {
		isFunction := kwIs(s.At(index), "function")
		if !isFunction && !kwIs(s.At(index), "fn") {
			continue
		}
		if !taIsLambda(s, index) {
			continue
		}
		if taKw(s, sigPrev(s, index), "static") {
			continue // already static
		}
		argumentsStart := nextPunctOfKind(s, index, "(")
		argumentsEnd := s.MatchForward(argumentsStart)
		if argumentsEnd < 0 {
			continue
		}
		var lambdaOpen, lambdaEnd int
		if isFunction {
			lambdaOpen = nextPunctOfKind(s, argumentsEnd, "{")
			if lambdaOpen < 0 {
				continue
			}
			lambdaEnd = s.MatchForward(lambdaOpen)
		} else {
			lambdaOpen = nextPunctOfKind(s, argumentsEnd, "=>")
			if lambdaOpen < 0 {
				continue
			}
			lambdaEnd = taArrowFunctionEnd(s, index)
		}
		if lambdaEnd < 0 {
			continue
		}
		if slHasPossibleReferenceToThis(s, lambdaOpen, lambdaEnd) {
			continue
		}
		if slIsDirectSubjectOfBinding(s, index, lambdaEnd) {
			continue // "(function () {})->bindTo($foo)" would warn once static
		}
		s.InsertSliceAt(index, []token.Token{
			{Kind: token.Keyword, Value: "static"},
			{Kind: token.Whitespace, Value: " "},
		})
		changed = true
		index -= 4 // the closest candidate is at least 4 tokens before
	}
	return changed
}

// slIsDirectSubjectOfBinding mirrors isDirectSubjectOfBinding: the lambda is
// alone in parentheses that are then the subject of ->bindTo( or ->call(.
func slIsDirectSubjectOfBinding(s *tokens.Stream, lambdaIndex, lambdaEnd int) bool {
	after := sigNext(s, lambdaEnd)
	if !isPunctVal(s, after, ")") {
		return false
	}
	wrapStart := s.MatchBackward(after)
	if wrapStart < 0 || sigNext(s, wrapStart) != lambdaIndex {
		return false // the parentheses contain more than the lambda
	}
	if before := sigPrev(s, wrapStart); before >= 0 {
		t := s.At(before)
		if isPunctVal(s, before, ")") || slIsIndexClose(s, before) || taIsTString(s, before) ||
			(t.Kind == token.Variable && t.Value != "$") {
			return false // an argument of a call, whose result is the subject
		}
	}
	op := sigNext(s, after)
	if !isPunctVal(s, op, "->") && !isPunctVal(s, op, "?->") {
		return false
	}
	method := sigNext(s, op)
	return taIsTString(s, method) && (strings.EqualFold(s.At(method).Value, "bindTo") || strings.EqualFold(s.At(method).Value, "call"))
}

// slIsIndexClose reports whether the "]" at i closes an index ("$a[0]"), which
// PHP keeps as "]"; a short-array close is CT::T_ARRAY_SQUARE_BRACE_CLOSE.
func slIsIndexClose(s *tokens.Stream, i int) bool {
	if !isPunctVal(s, i, "]") {
		return false
	}
	open := s.MatchBackward(i)
	return open >= 0 && !isArrayLiteralOpen(s, open)
}

// slHasPossibleReferenceToThis mirrors hasPossibleReferenceToThis over the
// inclusive range [start, end].
func slHasPossibleReferenceToThis(s *tokens.Stream, start, end int) bool {
	for i := start; i <= end && i < s.Len(); i++ {
		t := s.At(i)
		switch t.Kind {
		case token.Variable:
			if strings.ToLower(t.Value) == "$this" {
				return true
			}
			if t.Value == "$" {
				// "$$a" and "${$g}"
				if n := sigNext(s, i); n >= 0 && (s.At(n).Kind == token.Variable || isPunctVal(s, n, "{")) {
					return true
				}
			}
		case token.String:
			if slStringInterpolatesThis(t.Value) {
				return true
			}
		case token.Keyword:
			switch strings.ToLower(t.Value) {
			case "include", "include_once", "require", "require_once":
				return true
			}
		case token.Ident:
			if taIsConstruct(s, i, "eval") || strings.EqualFold(t.Value, "parent") {
				return true
			}
		}
		if taIsClassy(s, i) {
			openBrace := nextPunctOfKind(s, i, "{")
			if openBrace < 0 {
				return false
			}
			i = sigNext(s, i)
			if i >= 0 && i <= openBrace && slHasPossibleReferenceToThis(s, i, openBrace) {
				return true
			}
			i = s.MatchForward(openBrace)
			if i < 0 {
				return false
			}
		}
	}
	return false
}

// slStringInterpolatesThis reports whether a double-quoted string or heredoc
// token (or a chunk of one) interpolates $this, which PHP tokenizes as a
// T_VARIABLE inside the string. Single-quoted strings and nowdocs do not.
func slStringInterpolatesThis(v string) bool {
	switch {
	case strings.HasPrefix(v, "'"), strings.HasPrefix(v, "b'"), strings.HasPrefix(v, "B'"):
		return false
	case strings.HasPrefix(v, "<<<"):
		label := strings.TrimLeft(v[3:], " \t")
		if strings.HasPrefix(label, "'") {
			return false // nowdoc
		}
	}
	for i := 0; i < len(v); i++ {
		switch v[i] {
		case '\\':
			i++
		case '$':
			if i+5 <= len(v) && strings.EqualFold(v[i+1:i+5], "this") && (i+5 == len(v) || !slIsVarByte(v[i+5])) {
				return true
			}
		}
	}
	return false
}

func slIsVarByte(c byte) bool {
	return c == '_' || c >= 0x80 || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}
