package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/UseArrowFunctionsFixer.php
//
// UseArrowFunctions rewrites an anonymous function whose only statement is a
// return into an arrow function: "function ($a) use ($b) { return $a + $b; }"
// becomes "fn ($a) => $a + $b". Closures using variables by reference, or that
// include/require a file while importing variables, are left alone. Risky when
// isset() is used on outside variables not imported with use().
type UseArrowFunctions struct{}

func (UseArrowFunctions) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\UseArrowFunctionsFixer`
}

func (UseArrowFunctions) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/UseArrowFunctionsFixer.php"
}

func (UseArrowFunctions) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 1; index > 0; index-- {
		if s.At(index).Kind == token.AttributeEnd {
			if start := s.AttributeStartOf(index); start >= 0 {
				index = start
			}
			continue
		}
		if !kwIs(s.At(index), "function") || !taIsLambda(s, index) {
			continue
		}

		parametersStart := aliasNextMeaningful(s, index)
		if isPunctVal(s, parametersStart, "&") { // return by reference
			parametersStart = aliasNextMeaningful(s, parametersStart)
		}
		parametersEnd := s.MatchForward(parametersStart)
		if parametersEnd < 0 {
			continue
		}

		next := aliasNextMeaningful(s, parametersEnd)
		useStart, useEnd := -1, -1
		if kwIs(s.At(next), "use") {
			useStart = next
			if useStart-1 >= 0 && s.At(useStart-1).Kind == token.Whitespace {
				useStart--
			}
			next = aliasNextMeaningful(s, next)
			abort := false
			for !isPunctVal(s, next, ")") {
				if isPunctVal(s, next, "&") { // by-reference use is not supported
					abort = true
					break
				}
				next = aliasNextMeaningful(s, next)
				if next < 0 {
					abort = true
					break
				}
			}
			if abort {
				continue
			}
			useEnd = next
			next = aliasNextMeaningful(s, next)
		}

		braceOpen := next
		if !isPunctVal(s, next, "{") {
			braceOpen = nextPunctOfKind(s, next, "{")
		}
		if braceOpen < 0 {
			continue
		}
		returnIndex := braceOpen + 1
		if returnIndex < s.Len() && s.At(returnIndex).Kind == token.Whitespace {
			returnIndex++
		}
		if returnIndex >= s.Len() || !kwIs(s.At(returnIndex), "return") {
			continue
		}
		semicolon := nextPunctOfKind(s, returnIndex, "{", ";")
		if !isPunctVal(s, semicolon, ";") {
			continue
		}
		braceClose := semicolon + 1
		if braceClose < s.Len() && s.At(braceClose).Kind == token.Whitespace {
			braceClose++
		}
		if braceClose >= s.Len() || !isPunctVal(s, braceClose, "}") {
			continue
		}
		if useStart >= 0 && arrowContainsIncludeOrRequire(s, returnIndex, semicolon) {
			continue
		}

		inserts := []token.Token{{Kind: token.Punct, Value: "=>"}}
		if aliasNextMeaningful(s, returnIndex) == semicolon {
			inserts = append(inserts,
				token.Token{Kind: token.Whitespace, Value: " "},
				token.Token{Kind: token.Ident, Value: "null"})
		}
		arrowClearRange(s, semicolon, braceClose-1)
		clearTokenAndMergeWS(s, braceClose)
		arrowClearRange(s, braceOpen+1, returnIndex)
		s.ReplaceRange(braceOpen, braceOpen, inserts)
		if useStart >= 0 {
			arrowClearRange(s, useStart, useEnd)
		}
		s.Set(index, token.Token{Kind: token.Keyword, Value: "fn"})
		changed = true
	}
	return changed
}

// arrowContainsIncludeOrRequire reports whether an include/require construct
// appears in (start, end).
func arrowContainsIncludeOrRequire(s *tokens.Stream, start, end int) bool {
	for i := start; i < end; i++ {
		if kwIs(s.At(i), "include") || kwIs(s.At(i), "include_once") ||
			kwIs(s.At(i), "require") || kwIs(s.At(i), "require_once") {
			return true
		}
	}
	return false
}

// arrowClearRange clears every token in [start, end] to an empty placeholder.
func arrowClearRange(s *tokens.Stream, start, end int) {
	for i := start; i <= end && i < s.Len(); i++ {
		clearTokenAt(s, i)
	}
}
