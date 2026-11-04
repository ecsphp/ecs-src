package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/ArrayPushFixer.php
//
// ArrayPush converts a statement "array_push($x, $y);" to "$x[] = $y;" (risky:
// array_push may be overridden). Only a global array_push call that is a whole
// statement, pushes exactly one non-spread value and whose first argument is a
// plain variable-like expression is converted.
type ArrayPush struct{}

func (ArrayPush) Name() string {
	return `PhpCsFixer\Fixer\Alias\ArrayPushFixer`
}

func (ArrayPush) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/ArrayPushFixer.php"
}

func (ArrayPush) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 7; index > 0; index-- {
		if t := s.At(index); t.Kind != token.Ident || !strings.EqualFold(t.Value, "array_push") {
			continue
		}
		if !aliasIsGlobalFunctionCall(s, index) {
			continue // redeclare/override
		}

		// meaningful before must be `<?php`, `{`, `}`, `)` or `;`
		callIndex := index
		index = aliasPrevMeaningful(s, index)
		namespaceSeparatorIndex := -1
		if isPunctVal(s, index, `\`) {
			namespaceSeparatorIndex = index
			index = aliasPrevMeaningful(s, index)
		}
		if !isPunctVal(s, index, ";") && !isPunctVal(s, index, "{") && !isPunctVal(s, index, "}") && !isPunctVal(s, index, ")") && !aliasIsOpenTag(s, index) {
			continue
		}

		openBraceIndex := aliasNextMeaningful(s, callIndex)
		closeBraceIndex := s.MatchForward(openBraceIndex)
		if closeBraceIndex < 0 {
			continue
		}

		// meaningful after `)` must be `;`, `?>` or nothing
		if after := aliasNextMeaningful(s, closeBraceIndex); after >= 0 &&
			!isPunctVal(s, after, ";") && s.At(after).Kind != token.CloseTag {
			continue
		}

		// the first argument must be variable-like and followed by `,`
		firstArgumentStop := aliasNextMeaningful(s, arrayPushFirstArgumentEnd(s, openBraceIndex))
		if !isPunctVal(s, firstArgumentStop, ",") {
			continue
		}

		// the second argument is anything but a spread, and must be the last
		secondArgumentStart := aliasNextMeaningful(s, firstArgumentStop)
		if !arrayPushSecondArgumentIsLast(s, secondArgumentStart, closeBraceIndex) {
			continue
		}

		clearTokenAndMergeWS(s, closeBraceIndex)
		clearTokenAndMergeWS(s, firstArgumentStop)
		s.InsertSliceAt(firstArgumentStop, []token.Token{aliasPunct("["), aliasPunct("]"), aliasSpace(), aliasPunct("=")})
		clearTokenAndMergeWS(s, openBraceIndex)
		clearTokenAndMergeWS(s, callIndex)
		if namespaceSeparatorIndex >= 0 {
			clearTokenAndMergeWS(s, namespaceSeparatorIndex)
		}
		changed = true
	}
	if changed {
		aliasRelex(s)
	}
	return changed
}

// arrayPushFirstArgumentEnd ports ArrayPushFixer::getFirstArgumentEnd: the last
// token of the variable-like expression starting after index.
func arrayPushFirstArgumentEnd(s *tokens.Stream, index int) int {
	next := aliasNextMeaningful(s, index)
	for next >= 0 && arrayPushIsVariablePart(s, next) {
		if end := aliasBlockEnd(s, next); end >= 0 {
			next = end
		}
		index = next
		next = aliasNextMeaningful(s, next)
	}
	if isPunctVal(s, next, "->") {
		return arrayPushFirstArgumentEnd(s, next)
	}
	if isPunctVal(s, next, "::") {
		return arrayPushFirstArgumentEnd(s, aliasNextMeaningful(s, next))
	}
	return index
}

// arrayPushIsVariablePart reports the tokens getFirstArgumentEnd walks over:
// "$", an index "[", a plain "(", a dynamic "{", the namespace operator, "\",
// static, a name or a variable.
func arrayPushIsVariablePart(s *tokens.Stream, i int) bool {
	t := s.At(i)
	switch t.Kind {
	case token.Variable:
		return true // T_VARIABLE, or the "$" of a variable variable
	case token.Ident:
		return aliasIsTString(s, i)
	case token.Keyword:
		return kwIs(t, "static") || aliasIsNamespaceOperator(s, i)
	case token.Punct:
		switch t.Value {
		case `\`:
			return true
		case "[":
			return aliasIsIndexBracket(s, i)
		case "(":
			return aliasIsPlainParen(s, i)
		case "{":
			// CT::T_DYNAMIC_VAR_BRACE_OPEN "${" and CT::T_DYNAMIC_PROP_BRACE_OPEN "->{"
			p := aliasPrevMeaningful(s, i)
			return (p >= 0 && s.At(p).Kind == token.Variable && s.At(p).Value == "$") ||
				isPunctVal(s, p, "->") || isPunctVal(s, p, "?->")
		}
	}
	return false
}

// aliasIsIndexBracket reports whether the "[" at i is an index access (a plain
// "[" in PHP-CS-Fixer) rather than a short array or destructuring bracket, per
// the previous-token rule of its SquareBraceTransformer.
func aliasIsIndexBracket(s *tokens.Stream, i int) bool {
	p := aliasPrevMeaningful(s, i)
	if p < 0 {
		return false
	}
	t := s.At(p)
	switch t.Kind {
	case token.Punct:
		return t.Value == ")" || t.Value == "]" || t.Value == "}"
	case token.Variable:
		return t.Value != "$"
	case token.Ident:
		return aliasIsTString(s, p)
	case token.String:
		// a quoted literal or the closing quote of an interpolated string; not
		// a heredoc or backtick end
		v := t.Value
		return len(v) > 0 && (v[len(v)-1] == '\'' || v[len(v)-1] == '"') && !strings.HasPrefix(v, "<<<")
	}
	return false
}

// aliasIsPlainParen reports whether the "(" at i is a plain parenthesis: not a
// cast, and not the CT::T_BRACE_CLASS_INSTANTIATION_OPEN of "(new Foo)".
func aliasIsPlainParen(s *tokens.Stream, i int) bool {
	if _, _, ok := aliasCastAt(s, i); ok {
		return false
	}
	if !aliasIsKeyword(s, aliasNextMeaningful(s, i), "new") {
		return true
	}
	p := aliasPrevMeaningful(s, i)
	if p < 0 {
		return false
	}
	t := s.At(p)
	switch t.Kind {
	case token.Punct:
		return t.Value == "]"
	case token.Variable:
		return t.Value != "$"
	case token.Ident:
		return aliasIsTString(s, p)
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "array", "class", "elseif", "for", "foreach", "if", "static", "switch", "while":
			return true
		}
	}
	return false
}

// arrayPushSecondArgumentIsLast ports ArrayPushFixer::getSecondArgumentEnd: the
// second argument is not a spread and no further argument (nor a low-precedence
// yield/and/or/xor) follows it before endIndex.
func arrayPushSecondArgumentIsLast(s *tokens.Stream, index, endIndex int) bool {
	if index < 0 || aliasIsEllipsis(s, index) {
		return false
	}
	for ; index >= 0 && index <= endIndex; index++ {
		for index >= 0 {
			end := aliasBlockEnd(s, index)
			if end < 0 {
				break
			}
			index = aliasNextMeaningful(s, end)
		}
		if index < 0 {
			return false
		}
		if isPunctVal(s, index, ",") {
			return false
		}
		if t := s.At(index); t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "yield", "and", "or", "xor":
				return false
			}
		}
	}
	return true
}
