package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/OperatorLinebreakFixer.php
//
// OperatorLinebreak moves a multiline operator to the beginning of the next line
// (default position). It handles the full operator set, including ":", "?", "|"
// and "&", by classifying those ambiguous tokens the way PHP-CS-Fixer's
// tokenizer transformers + analyzers do: a ":" that is a type-colon, named
// argument, switch case, goto label or alternative-syntax colon is left alone; a
// "?" that is a nullable type marker is left alone; a "|"/"&" that is a union or
// intersection type separator is left alone; and a "&" that is a return-ref or a
// reference is left alone. Object operators "->"/"?->" and "::" are moved too,
// matching php-cs-fixer's getObjectOperatorKinds() + T_PAAMAYIM_NEKUDOTAYIM.
type OperatorLinebreak struct{}

// operatorLinebreakPunct are the always-unambiguous operator tokens.
var operatorLinebreakPunct = map[string]bool{
	"||": true, "&&": true, ".": true, "+": true, "-": true, "*": true,
	"/": true, "%": true, "**": true, "==": true, "===": true, "!=": true,
	"!==": true, "<>": true, "<": true, ">": true, "<=": true, ">=": true,
	"<=>": true, "??": true, "=>": true, "=": true, ".=": true, "+=": true,
	"-=": true, "*=": true, "/=": true, "%=": true, "**=": true, "&=": true,
	"|=": true, "^=": true, "<<=": true, ">>=": true, "??=": true, "^": true,
	"<<": true, ">>": true, "|": true, "&": true, ":": true, "?": true,
	"->": true, "?->": true, "::": true,
}

func (OperatorLinebreak) Name() string {
	return `PhpCsFixer\Fixer\Operator\OperatorLinebreakFixer`
}

func (OperatorLinebreak) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/OperatorLinebreakFixer.php"
}

func isOperatorLinebreakToken(t token.Token) bool {
	switch t.Kind {
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "and", "or", "xor":
			return true
		}
	case token.Punct:
		return operatorLinebreakPunct[t.Value]
	}
	return false
}

// operatorLinebreakExcluded reports whether the token at i, although in the
// operator set, must be left alone because of its context (mirrors what the
// PHP-CS-Fixer tokenizer transforms into a CT token, plus the fixer's own
// goto-label / reference / alternative-syntax / switch guards).
func operatorLinebreakExcluded(s *tokens.Stream, i int, switchColons map[int]bool) bool {
	if s.At(i).Kind != token.Punct {
		return false
	}
	switch s.At(i).Value {
	case ":":
		return isTypeColon(s, i) || isNamedArgumentColon(s, i) ||
			belongsToGotoLabel(s, i) || belongsToAlternativeSyntax(s, i) || switchColons[i]
	case "?":
		return isNullableType(s, i)
	case "|":
		return isPartOfType(s, i)
	case "&":
		return isPartOfType(s, i) || isReturnRef(s, i) || isReferenceAmp(s, i)
	}
	return false
}

func (OperatorLinebreak) Fix(s *tokens.Stream) bool {
	changed := false
	switchColons := switchCaseColons(s)
	for i := s.Len() - 1; i > 0; i-- {
		if !isOperatorLinebreakToken(s.At(i)) {
			continue
		}
		if operatorLinebreakExcluded(s, i, switchColons) {
			continue
		}

		opIndices := []int{i}
		// short-ternary "?:" is moved as a single unit
		if s.At(i).Kind == token.Punct && s.At(i).Value == ":" {
			if p := sigPrev(s, i); p >= 0 && s.At(p).Kind == token.Punct && s.At(p).Value == "?" {
				opIndices = []int{p, i}
			}
		}

		lo, hi := opIndices[0], opIndices[len(opIndices)-1]
		prevM := sigPrev(s, lo)
		nextM := sigNext(s, hi)
		if prevM < 0 || nextM < 0 {
			continue
		}
		// operator must be surrounded by multiline whitespace
		if !operatorSpanMultiline(s, prevM+1, nextM-1) {
			continue
		}
		// already at the beginning of a line if there is no newline after it
		if !operatorSpanMultiline(s, hi, nextM-1) {
			continue
		}
		moveOperatorToLineStart(s, opIndices, nextM)
		changed = true
		i = lo // resume before the moved group
	}
	return changed
}

// operatorSpanMultiline mirrors OperatorLinebreak::isMultiline - whether any
// token in the inclusive range [from, to] contains a newline.
func operatorSpanMultiline(s *tokens.Stream, from, to int) bool {
	if from < 0 {
		from = 0
	}
	for j := from; j <= to && j < s.Len(); j++ {
		if strings.ContainsAny(s.At(j).Value, "\n\r") {
			return true
		}
	}
	return false
}

// moveOperatorToLineStart mirrors fixMoveToTheBeginning: it clears each operator
// (and the whitespace immediately before it) and re-inserts the operators just
// before the next meaningful token, keeping a single leading space when a space
// preceded the group.
func moveOperatorToLineStart(s *tokens.Stream, opIndices []int, nextM int) {
	lo := opIndices[0]
	hadSpaceBefore := lo > 0 && s.At(lo-1).Kind == token.Whitespace

	// clone operator tokens and clear them plus the whitespace directly before each
	clones := make([]token.Token, len(opIndices))
	for k, idx := range opIndices {
		clones[k] = token.Token{Kind: s.At(idx).Kind, Value: s.At(idx).Value}
	}
	for _, idx := range opIndices {
		if idx > 0 && s.At(idx-1).Kind == token.Whitespace {
			s.SetValue(idx-1, "")
		}
		s.SetValue(idx, "")
	}

	s.InsertSliceAt(nextM, clones)
	at := nextM + len(clones)
	if hadSpaceBefore {
		s.InsertAt(at, token.Token{Kind: token.Whitespace, Value: " "})
	}
}
