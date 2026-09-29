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
	if s.At(prevIndex).Kind == token.Ident {
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

// isNamedArgumentColon mirrors NamedArgumentTransformer.
func isNamedArgumentColon(s *tokens.Stream, index int) bool {
	if !isPunctVal(s, index, ":") {
		return false
	}
	stringIndex := sigPrev(s, index)
	if stringIndex < 0 || s.At(stringIndex).Kind != token.Ident {
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
	ok := isPunctVal(s, prevIndex, "(") || isPunctVal(s, prevIndex, ",") || isTypeColon(s, prevIndex)
	if !ok {
		switch olKwLower(s.At(prevIndex)) {
		case "public", "protected", "private", "var", "static", "const", "abstract", "final", "readonly":
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

// switchCaseColons returns the set of ":" indices that terminate a case/default
// label inside a switch (SwitchAnalyzer::belongsToSwitch).
func switchCaseColons(s *tokens.Stream) map[int]bool {
	result := map[int]bool{}
	for i := 0; i < s.Len(); i++ {
		if !kwIs(s.At(i), "switch") {
			continue
		}
		open := nextPunctOfKind(s, i, "(")
		if open < 0 {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 {
			continue
		}
		bodyStart := sigNext(s, closeIdx)
		if bodyStart < 0 {
			continue
		}
		if isPunctVal(s, bodyStart, "{") {
			bodyEnd := s.MatchForward(bodyStart)
			if bodyEnd < 0 {
				continue
			}
			scanCaseColons(s, bodyStart+1, bodyEnd, result)
		} else if isPunctVal(s, bodyStart, ":") {
			result[bodyStart] = true
			if end := altSwitchEnd(s, bodyStart); end > bodyStart {
				scanCaseColons(s, bodyStart+1, end, result)
			}
		}
	}
	return result
}

func scanCaseColons(s *tokens.Stream, from, to int, result map[int]bool) {
	depth := 0
	for j := from; j < to && j < s.Len(); j++ {
		if isPunctVal(s, j, "{") {
			depth++
			continue
		}
		if isPunctVal(s, j, "}") {
			depth--
			continue
		}
		if depth == 0 && (kwIs(s.At(j), "case") || kwIs(s.At(j), "default")) {
			if c := caseColonAfter(s, j, to); c >= 0 {
				result[c] = true
			}
		}
	}
}

// caseColonAfter finds the ":" (or ";") terminating a case/default label.
func caseColonAfter(s *tokens.Stream, caseIdx, limit int) int {
	depth := 0
	for j := caseIdx + 1; j < limit && j < s.Len(); j++ {
		if s.At(j).Kind != token.Punct {
			continue
		}
		switch s.At(j).Value {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		case ":", ";":
			if depth == 0 {
				if s.At(j).Value == ":" {
					return j
				}
				return -1
			}
		}
	}
	return -1
}

// altSwitchEnd finds the endswitch terminating an alternative-syntax switch body.
func altSwitchEnd(s *tokens.Stream, colonIndex int) int {
	depth := 0
	for j := colonIndex + 1; j < s.Len(); j++ {
		if kwIs(s.At(j), "switch") {
			depth++
			continue
		}
		if kwIs(s.At(j), "endswitch") {
			if depth == 0 {
				return j
			}
			depth--
		}
	}
	return -1
}
