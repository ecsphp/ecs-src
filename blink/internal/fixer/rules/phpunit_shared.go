package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Helpers shared by the AbstractPhpUnitFixer ports (PhpUnitConstruct,
// PhpUnitFqcnAnnotation, PhpUnitMockShortWillReturn, PhpUnitTestAnnotation).

// puClassesLastFirst mirrors PhpUnitTestCaseAnalyzer::findPhpUnitClasses: the
// [open brace, close brace] of every PHPUnit test class, last class first, so a
// fixer that inserts tokens never shifts a class it has yet to visit.
func puClassesLastFirst(s *tokens.Stream) [][2]int {
	classes := phpUnitTestClasses(s)
	for i, j := 0, len(classes)-1; i < j; i, j = i+1, j-1 {
		classes[i], classes[j] = classes[j], classes[i]
	}
	return classes
}

// puIsObjectOperator mirrors Token::isObjectOperator ("->" or "?->").
func puIsObjectOperator(s *tokens.Stream, i int) bool {
	return isPunctVal(s, i, "->") || isPunctVal(s, i, "?->")
}

// puIsTheSameClassCall mirrors FunctionsAnalyzer::isTheSameClassCall: the name
// at index is called as $this->name(, self::name( or static::name( (the
// receiver compared case-insensitively).
func puIsTheSameClassCall(s *tokens.Stream, index int) bool {
	if index < 0 || index >= s.Len() {
		return false
	}
	op := sigPrev(s, index)
	if op < 0 || !(puIsObjectOperator(s, op) || isPunctVal(s, op, "::")) {
		return false
	}
	ref := sigPrev(s, op)
	if ref < 0 {
		return false
	}
	t := s.At(ref)
	switch {
	case t.Kind == token.Variable && strings.EqualFold(t.Value, "$this"):
	case t.Kind == token.Ident && strings.EqualFold(t.Value, "self"):
	case t.Kind == token.Keyword && strings.EqualFold(t.Value, "static"):
	default:
		return false
	}
	return isPunctVal(s, sigNext(s, index), "(")
}

// puIsNativeConstant mirrors Token::isNativeConstant: true, false or null in any
// case (blink lexes them as identifiers, PHP as T_STRING).
func puIsNativeConstant(t token.Token) bool {
	if t.Kind != token.Ident {
		return false
	}
	switch strings.ToLower(t.Value) {
	case "true", "false", "null":
		return true
	}
	return false
}

// puNextNonWhitespace mirrors Tokens::getNextNonWhitespace (cleared placeholder
// tokens count as whitespace, as PHP's empty tokens do).
func puNextNonWhitespace(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		if s.At(j).Kind != token.Whitespace {
			return j
		}
	}
	return -1
}
