package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Context classifiers for OperatorLinebreak. Each mirrors a PHP-CS-Fixer
// tokenizer transformer or analyzer so an ambiguous ":" / "?" / "|" / "&" is
// recognised as a type / nullable / union / intersection / reference / label /
// switch / alternative-syntax context and left untouched.

func olKwLower(t token.Token) string {
	if t.Kind != token.Keyword {
		return ""
	}
	return strings.ToLower(t.Value)
}

// isTypeColon mirrors TypeColonTransformer: a ":" that introduces a return type
// (or an enum backing type).
func isTypeColon(s *tokens.Stream, index int) bool {
	if !isPunctVal(s, index, ":") {
		return false
	}
	endIndex := sigPrev(s, index)
	if endIndex < 0 {
		return false
	}
	if pe := sigPrev(s, endIndex); pe >= 0 && kwIs(s.At(pe), "enum") {
		return true
	}
	if !isPunctVal(s, endIndex, ")") {
		return false
	}
	startIndex := s.MatchBackward(endIndex)
	if startIndex < 0 {
		return false
	}
	prevIndex := sigPrev(s, startIndex)
	if prevIndex < 0 {
		return false
	}
	if t := s.At(prevIndex); t.Kind == token.Ident || (t.Kind == token.Keyword && isFunctionNameKeyword(s, prevIndex)) {
		prevIndex = sigPrev(s, prevIndex)
		if prevIndex < 0 {
			return false
		}
	}
	switch olKwLower(s.At(prevIndex)) {
	case "function", "fn", "use":
		return true
	}
	return isPunctVal(s, prevIndex, "&") && isReturnRef(s, prevIndex)
}

// isFunctionNameKeyword reports whether the reserved word at index is a
// function name ("function list()", "function &print()"): PHP-CS-Fixer
// tokenizes with TOKEN_PARSE, which makes it a T_STRING there.
func isFunctionNameKeyword(s *tokens.Stream, index int) bool {
	prev := sigPrev(s, index)
	if prev < 0 {
		return false
	}
	return kwIs(s.At(prev), "function") || isReturnRef(s, prev)
}

// isNamedArgumentColon mirrors NamedArgumentTransformer. PHP-CS-Fixer
// tokenizes with TOKEN_PARSE, so a reserved word used as an argument name
// ("f(default: 1)", "new A(class: B::class)") is a T_STRING there too.
func isNamedArgumentColon(s *tokens.Stream, index int) bool {
	if !isPunctVal(s, index, ":") {
		return false
	}
	stringIndex := sigPrev(s, index)
	if stringIndex < 0 || (s.At(stringIndex).Kind != token.Ident && s.At(stringIndex).Kind != token.Keyword) {
		return false
	}
	preString := sigPrev(s, stringIndex)
	if preString < 0 {
		return false
	}
	return isPunctVal(s, preString, ",") || isPunctVal(s, preString, "(")
}

// isNullableType mirrors NullableTypeTransformer.
func isNullableType(s *tokens.Stream, index int) bool {
	if !isPunctVal(s, index, "?") {
		return false
	}
	prevIndex := sigPrev(s, index)
	if prevIndex < 0 {
		return false
	}
	ok := isPunctVal(s, prevIndex, "(") || isPunctVal(s, prevIndex, ",") || isTypeColon(s, prevIndex) ||
		s.Kind(prevIndex) == token.AttributeEnd
	if !ok {
		switch olKwLower(s.At(prevIndex)) {
		case "public", "protected", "private", "var", "static", "const", "abstract", "final", "readonly",
			"public(set)", "protected(set)", "private(set)":
			ok = true
		}
	}
	if !ok {
		return false
	}
	if olKwLower(s.At(prevIndex)) == "static" {
		if pp := sigPrev(s, prevIndex); pp >= 0 && kwIs(s.At(pp), "instanceof") {
			return false
		}
	}
	return true
}

// isReturnRef mirrors ReturnRefTransformer: "&" right after function/fn.
func isReturnRef(s *tokens.Stream, index int) bool {
	if !isPunctVal(s, index, "&") {
		return false
	}
	prev := sigPrev(s, index)
	if prev < 0 {
		return false
	}
	return kwIs(s.At(prev), "function") || kwIs(s.At(prev), "fn")
}

// isReferenceAmp mirrors ReferenceAnalyzer::isReference (the "&" is a reference,
// not bitwise-and), excluding the return-ref case handled separately.
func isReferenceAmp(s *tokens.Stream, index int) bool {
	if !isPunctVal(s, index, "&") {
		return false
	}
	idx := sigPrev(s, index)
	if idx < 0 {
		return false
	}
	t := s.At(idx)
	if isPunctVal(s, idx, "=") || isPunctVal(s, idx, "=>") ||
		kwIs(t, "as") || kwIs(t, "callable") || kwIs(t, "array") {
		return true
	}
	if t.Kind == token.Ident {
		idx = sigPrev(s, idx)
		if idx < 0 {
			return false
		}
	}
	return isPunctVal(s, idx, "(") || isPunctVal(s, idx, ",") || isPunctVal(s, idx, `\`) ||
		(isPunctVal(s, idx, "?") && isNullableType(s, idx))
}

// belongsToGotoLabel mirrors GotoLabelAnalyzer.
func belongsToGotoLabel(s *tokens.Stream, index int) bool {
	if !isPunctVal(s, index, ":") {
		return false
	}
	prev := sigPrev(s, index)
	if prev < 0 || s.At(prev).Kind != token.Ident {
		return false
	}
	prev2 := sigPrev(s, prev)
	if prev2 < 0 {
		return false
	}
	return isPunctVal(s, prev2, ":") || isPunctVal(s, prev2, ";") ||
		isPunctVal(s, prev2, "{") || isPunctVal(s, prev2, "}") || s.At(prev2).Kind == token.OpenTag
}

// belongsToAlternativeSyntax mirrors AlternativeSyntaxAnalyzer.
func belongsToAlternativeSyntax(s *tokens.Stream, index int) bool {
	if !isPunctVal(s, index, ":") {
		return false
	}
	prev := sigPrev(s, index)
	if prev < 0 {
		return false
	}
	if kwIs(s.At(prev), "else") {
		return true
	}
	if !isPunctVal(s, prev, ")") {
		return false
	}
	open := s.MatchBackward(prev)
	if open < 0 {
		return false
	}
	before := sigPrev(s, open)
	if before < 0 {
		return false
	}
	switch olKwLower(s.At(before)) {
	case "declare", "elseif", "for", "foreach", "if", "switch", "while":
		return true
	}
	return false
}

// typeSkipToken reports whether j is one of AbstractTypeTransformer::TYPE_TOKENS.
func typeSkipToken(s *tokens.Stream, j int) bool {
	t := s.At(j)
	switch t.Kind {
	case token.Whitespace, token.Comment, token.DocComment, token.Ident:
		return true
	case token.Punct:
		switch t.Value {
		case "|", "&", "(", ")", `\`:
			return true
		}
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "callable", "static", "array":
			return true
		}
	}
	return false
}

// notOfKindSiblingType walks from index in direction dir, skipping TYPE_TOKENS.
func notOfKindSiblingType(s *tokens.Stream, index, dir int) int {
	for j := index + dir; j >= 0 && j < s.Len(); j += dir {
		if !typeSkipToken(s, j) {
			return j
		}
	}
	return -1
}

func isTypeEndToken(s *tokens.Stream, idx int) bool {
	if isPunctVal(s, idx, ")") || isPunctVal(s, idx, `\`) {
		return true
	}
	t := s.At(idx)
	if t.Kind == token.Ident {
		return true
	}
	switch olKwLower(t) {
	case "callable", "static", "array":
		return true
	}
	return false
}

// isPartOfType mirrors AbstractTypeTransformer::isPartOfType for a "|"/"&".
func isPartOfType(s *tokens.Stream, index int) bool {
	if !isPunctVal(s, index, "|") && !isPunctVal(s, index, "&") {
		return false
	}
	if typeColonIndex := notOfKindSiblingType(s, index, -1); typeColonIndex >= 0 {
		tc := s.At(typeColonIndex)
		if kwIs(tc, "catch") || kwIs(tc, "const") || isTypeColon(s, typeColonIndex) {
			return true
		}
	}
	afterTypeIndex := notOfKindSiblingType(s, index, 1)
	if afterTypeIndex < 0 {
		return false
	}
	if isPunctVal(s, afterTypeIndex, "...") {
		return true
	}
	if s.At(afterTypeIndex).Kind != token.Variable {
		return false
	}
	beforeVar := sigPrev(s, afterTypeIndex)
	if beforeVar < 0 {
		return false
	}
	if isPunctVal(s, beforeVar, "&") {
		prevIndex := getPrevTokenOfKindType(s, index)
		return prevIndex >= 0 && (kwIs(s.At(prevIndex), "fn") || kwIs(s.At(prevIndex), "function"))
	}
	return isTypeEndToken(s, beforeVar)
}

// getPrevTokenOfKindType scans back for "{", "}", ";", a close tag, or fn/function.
func getPrevTokenOfKindType(s *tokens.Stream, index int) int {
	for j := index - 1; j >= 0; j-- {
		if isPunctVal(s, j, "{") || isPunctVal(s, j, "}") || isPunctVal(s, j, ";") ||
			s.At(j).Kind == token.CloseTag || kwIs(s.At(j), "fn") || kwIs(s.At(j), "function") {
			return j
		}
	}
	return -1
}

// switchCaseColons returns the set of ":" indices that open a switch body or
// terminate a case/default label (SwitchAnalyzer::belongsToSwitch). It ports
// ControlCaseStructuresAnalyzer::findControlStructures: switch, match and enum
// are stacked, each counting its own braces, so a brace-style switch nested in
// an alternative-syntax one (or a match "default" inside a switch) is scoped to
// its own structure.
func switchCaseColons(s *tokens.Stream) map[int]bool {
	type caseStructure struct {
		kind        string
		braces      int
		alternative bool
	}
	result := map[int]bool{}
	var stack []caseStructure
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if kind := caseStructureKind(s, i); kind != "" {
			cs := caseStructure{kind: kind}
			if kind == "switch" {
				if open := sigNext(s, i); isPunctVal(s, open, "(") {
					if c := s.MatchForward(open); c >= 0 {
						if body := sigNext(s, c); isPunctVal(s, body, ":") {
							cs.alternative = true
							result[body] = true
						}
					}
				}
			}
			stack = append(stack, cs)
			continue
		}
		if len(stack) == 0 {
			continue
		}
		top := &stack[len(stack)-1]
		switch {
		case isPunctVal(s, i, "{"):
			top.braces++
			continue
		case isPunctVal(s, i, "}"):
			top.braces--
			if top.braces == 0 && !top.alternative {
				stack = stack[:len(stack)-1]
			}
			continue
		case kwIs(t, "endswitch"):
			stack = stack[:len(stack)-1]
			continue
		}
		if top.kind != "switch" {
			continue
		}
		if kwIs(t, "case") {
			if c := caseColonAfter(s, i); c >= 0 {
				result[c] = true
			}
		} else if kwIs(t, "default") {
			if c := nextPunctOfKind(s, i, ":", ";"); isPunctVal(s, c, ":") {
				result[c] = true
			}
		}
	}
	return result
}

// caseStructureKind reports whether index starts a switch, match or enum, the
// structures ControlCaseStructuresAnalyzer tracks. "enum" is T_ENUM only when a
// name follows it.
func caseStructureKind(s *tokens.Stream, index int) string {
	t := s.At(index)
	if t.Kind != token.Keyword {
		return ""
	}
	switch kw := strings.ToLower(t.Value); kw {
	case "switch", "match":
		return kw
	case "enum":
		if n := sigNext(s, index); n >= 0 && s.At(n).Kind == token.Ident {
			return kw
		}
	}
	return ""
}

// caseColonAfter ports ControlCaseStructuresAnalyzer::findCaseOpen: the ":"
// (or ";") ending a case label, skipping "(...)" and "{...}" blocks; a ternary
// "?" in the case expression consumes the next ":" (or ";"). It returns -1 when
// the label ends with ";".
func caseColonAfter(s *tokens.Stream, caseIdx int) int {
	ternaries := 0
	for j := caseIdx + 1; j < s.Len(); j++ {
		if s.At(j).Kind != token.Punct {
			continue
		}
		switch s.At(j).Value {
		case "(", "{":
			if end := s.MatchForward(j); end > j {
				j = end
			}
		case "?":
			ternaries++
		case ":", ";":
			if ternaries > 0 {
				ternaries--
				continue
			}
			if s.At(j).Value == ":" {
				return j
			}
			return -1
		}
	}
	return -1
}
