package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Symplify: https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/MethodChainingNewlineFixer.php
//
// MethodChainingNewline puts each chained "->" method call on its own line by
// inserting a line break before the operator. It fires only for a "->" whose
// receiver is a call that ends in ")" (not a plain variable or "new"), and
// skips chains inside a boolean/comparison expression, a control condition, a
// trailing short no-arg accessor, a method-call/array argument, a function call
// receiver, or an already multiline call. Only "->" is handled, not "?->", as
// upstream keys on T_OBJECT_OPERATOR alone.
type MethodChainingNewline struct{}

func (MethodChainingNewline) Name() string {
	return `Symplify\CodingStandard\Fixer\Spacing\MethodChainingNewlineFixer`
}

func (MethodChainingNewline) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Spacing/MethodChainingNewlineFixer.php"
}

func (MethodChainingNewline) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 1; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || t.Value != "->" {
			continue
		}
		if !mcnShouldPrefixNewline(s, i) {
			continue
		}
		s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: "\n"})
		changed = true
		i++ // skip the "->" now shifted to i+1
	}
	return changed
}

// mcnShouldPrefixNewline walks back from the "->" to the token that decides the
// chain: a closing ")" defers to mcnShouldBracketPrefix, while "new", a variable
// or an opening "(" means no newline.
func mcnShouldPrefixNewline(s *tokens.Stream, objectOperatorIndex int) bool {
	for i := objectOperatorIndex; i >= 0; i-- {
		t := s.At(i)
		if t.Kind == token.Punct && t.Value == ")" {
			return mcnShouldBracketPrefix(s, i, objectOperatorIndex)
		}
		if t.Kind == token.Variable || kwIs(t, "new") {
			return false
		}
		if t.Kind == token.Punct && t.Value == "(" {
			return false
		}
	}
	return false
}

func mcnShouldBracketPrefix(s *tokens.Stream, position, objectOperatorIndex int) bool {
	if mcnIsDoubleBracket(s, position) {
		return false
	}
	if mcnIsPartOfBooleanOrComparison(s, objectOperatorIndex) {
		return false
	}
	if mcnIsInsideControlCondition(s, objectOperatorIndex) {
		return false
	}
	if mcnIsShortNoArgTrailingMethod(s, objectOperatorIndex) {
		return false
	}
	if mcnIsPartOfMethodCallOrArray(s, position) {
		return false
	}
	if mcnIsPrecededByFuncCall(s, position) {
		return false
	}
	if mcnIsPrecededByOpenedCallInAnotherBracket(s, position) {
		return false
	}
	// all good, the chain is on one line and may be broken
	return !mcnPartialCodeMultiline(s, position, objectOperatorIndex)
}

// mcnIsDoubleBracket reports "())" - a call whose ")" is directly followed by a
// second ")", so the "->" belongs to the outer call, not this one.
func mcnIsDoubleBracket(s *tokens.Stream, position int) bool {
	n := nextSignificantIndex(s, position)
	return isPunctVal(s, n, ")")
}

// mcnIsPrecededByFuncCall matches e.g. clone, return app()->some(), where the
// receiver is a function call or a clone expression that must stay on its line.
func mcnIsPrecededByFuncCall(s *tokens.Stream, position int) bool {
	for i := position; i >= 0; i-- {
		t := s.At(i)
		if t.Value == "clone" {
			return true
		}
		if t.Kind == token.Punct && t.Value == "(" {
			return mcnDoesContentBeforeBracketRequireNewline(s, i)
		}
		if mcnIsNewlineToken(t) {
			return false
		}
	}
	return false
}

func mcnDoesContentBeforeBracketRequireNewline(s *tokens.Stream, i int) bool {
	prev := prevSignificantIndex(s, i)
	if prev < 0 || s.At(prev).Kind != token.Ident {
		return false
	}
	pp := prevSignificantIndex(s, prev)
	if pp < 0 {
		return false
	}
	ppTok := s.At(pp)
	if ppTok.Value == "{" {
		return true
	}
	// a function body: return, "::" or "{"
	return kwIs(ppTok, "return") || (ppTok.Kind == token.Punct && ppTok.Value == "::")
}

// mcnIsPartOfMethodCallOrArray matches someMethod($this->a()->b()) or
// [$this->a()->b()]: a chain nested in another call's arguments or an array.
func mcnIsPartOfMethodCallOrArray(s *tokens.Stream, position int) bool {
	bracketNesting := 0
	for i := position; i >= 0; i-- {
		t := s.At(i)
		if mcnIsNewlineToken(t) {
			return false
		}
		if mcnIsBreakingChar(t) {
			return true
		}
		if mcnShouldBreakOnBracket(t, &bracketNesting) {
			return true
		}
	}
	return false
}

func mcnIsBreakingChar(t token.Token) bool {
	if kwIs(t, "array") {
		return true
	}
	if t.Kind == token.Punct {
		return t.Value == "[" || t.Value == "::" || t.Value == "."
	}
	return false
}

func mcnShouldBreakOnBracket(t token.Token, bracketNesting *int) bool {
	if t.Kind != token.Punct {
		return false
	}
	if t.Value == ")" {
		*bracketNesting--
		return false
	}
	if t.Value == "(" {
		if *bracketNesting != 0 {
			*bracketNesting++
			return false
		}
		return true
	}
	return false
}

// mcnIsPartOfBooleanOrComparison matches $x->a()->b() || ... or
// $x->a()->b() === Foo::class on either side of the chain.
func mcnIsPartOfBooleanOrComparison(s *tokens.Stream, position int) bool {
	return mcnHasOperatorInDirection(s, position, -1) || mcnHasOperatorInDirection(s, position, 1)
}

func mcnHasOperatorInDirection(s *tokens.Stream, position, step int) bool {
	bracketNesting := 0
	for i := position + step; i >= 0 && i < s.Len(); i += step {
		t := s.At(i)
		content := t.Value
		enter := (step < 0 && (content == ")" || content == "]")) ||
			(step > 0 && (content == "(" || content == "["))
		if t.Kind == token.Punct && enter {
			bracketNesting++
			continue
		}
		leave := (step < 0 && (content == "(" || content == "[")) ||
			(step > 0 && (content == ")" || content == "]"))
		if t.Kind == token.Punct && leave {
			if bracketNesting == 0 {
				return false
			}
			bracketNesting--
			continue
		}
		if bracketNesting != 0 {
			continue
		}
		if t.Kind == token.Punct && (content == ";" || content == "{" || content == "}") {
			return false
		}
		if mcnIsBooleanOrComparisonToken(t) {
			return true
		}
	}
	return false
}

func mcnIsBooleanOrComparisonToken(t token.Token) bool {
	switch t.Kind {
	case token.Punct:
		switch t.Value {
		case "&&", "||", "==", "!=", "<>", "===", "!==", "<=", ">=", "<=>", "<", ">":
			return true
		}
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "and", "or", "xor":
			return true
		}
	}
	return false
}

// mcnIsInsideControlCondition matches if ($x->a()->b()), while (...), etc.: a
// chain inside the parentheses of a control structure.
func mcnIsInsideControlCondition(s *tokens.Stream, position int) bool {
	bracketNesting := 0
	for i := position; i >= 0; i-- {
		t := s.At(i)
		content := t.Value
		if t.Kind == token.Punct && (content == ")" || content == "]") {
			bracketNesting++
			continue
		}
		if t.Kind == token.Punct && (content == "(" || content == "[") {
			if bracketNesting != 0 {
				bracketNesting--
				continue
			}
			if content == "[" {
				return false
			}
			before := prevMeaningfulIndex(s, i)
			if before < 0 {
				return false
			}
			bt := s.At(before)
			return kwIs(bt, "if") || kwIs(bt, "elseif") || kwIs(bt, "while") || kwIs(bt, "switch")
		}
		if bracketNesting == 0 && t.Kind == token.Punct && (content == ";" || content == "{" || content == "}") {
			return false
		}
	}
	return false
}

// mcnIsShortNoArgTrailingMethod matches a trailing accessor like ->yes() whose
// name is at most five characters and takes no arguments.
func mcnIsShortNoArgTrailingMethod(s *tokens.Stream, objectOperatorIndex int) bool {
	name := nextMeaningfulIndex(s, objectOperatorIndex)
	if name < 0 || s.At(name).Kind != token.Ident {
		return false
	}
	if len(s.At(name).Value) > 5 {
		return false
	}
	open := nextMeaningfulIndex(s, name)
	if open < 0 || !isPunctVal(s, open, "(") {
		return false
	}
	closeIdx := nextMeaningfulIndex(s, open)
	return isPunctVal(s, closeIdx, ")")
}

// mcnIsPrecededByOpenedCallInAnotherBracket matches app([ ])->some(): the
// receiver's own "(...)" group already spans several lines.
func mcnIsPrecededByOpenedCallInAnotherBracket(s *tokens.Stream, position int) bool {
	open := s.MatchBackward(position)
	if open < 0 {
		return false
	}
	return mcnPartialCodeMultiline(s, open, position)
}

func mcnIsNewlineToken(t token.Token) bool {
	return t.Kind == token.Whitespace && strings.Contains(t.Value, "\n")
}

// mcnPartialCodeMultiline reports whether any token in [start, end] holds a
// newline, mirroring Tokens::isPartialCodeMultiline.
func mcnPartialCodeMultiline(s *tokens.Stream, start, end int) bool {
	for i := start; i <= end; i++ {
		if strings.Contains(s.At(i).Value, "\n") {
			return true
		}
	}
	return false
}
