package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// A partial port of PHP-CS-Fixer's CommentsAnalyzer, enough for PhpdocToComment:
// deciding whether a doc comment is a file header or sits before a structural
// element (a declaration, control statement or documented variable).

var structuralSkipKeywords = map[string]bool{
	"private": true, "protected": true, "public": true, "var": true,
	"function": true, "fn": true, "abstract": true, "const": true,
	"namespace": true, "require": true, "require_once": true,
	"include": true, "include_once": true, "final": true, "readonly": true,
}

var classyKeywords = map[string]bool{
	"class": true, "interface": true, "trait": true, "enum": true,
}

var assignmentOps = map[string]bool{
	"=": true, "+=": true, "-=": true, "*=": true, "/=": true, "%=": true,
	"**=": true, "&=": true, "|=": true, "^=": true, "<<=": true, ">>=": true,
	"??=": true, ".=": true,
}

func docIsHeaderComment(s *tokens.Stream, index int) bool {
	if sigNext(s, index) < 0 {
		return false
	}
	prev := getPrevNonWhitespace(s, index)
	if prev < 0 {
		return false
	}
	if isPunctVal(s, prev, ";") {
		braceClose := sigPrev(s, prev)
		if !isPunctVal(s, braceClose, ")") {
			return false
		}
		braceOpen := s.MatchBackward(braceClose)
		if braceOpen < 0 {
			return false
		}
		declare := sigPrev(s, braceOpen)
		if declare < 0 || !kwIs(s.At(declare), "declare") {
			return false
		}
		prev = getPrevNonWhitespace(s, declare)
		if prev < 0 {
			return false
		}
	}
	return s.At(prev).Kind == token.OpenTag
}

// docNextTokenIndex mirrors CommentsAnalyzer::getNextTokenIndex - the next
// meaningful token, skipping attributes (already comments here) and stepping
// past a leading "(".
func docNextTokenIndex(s *tokens.Stream, index int) int {
	next := sigNext(s, index)
	for next >= 0 && isPunctVal(s, next, "(") {
		next = sigNext(s, next)
	}
	return next
}

func docIsBeforeStructuralElement(s *tokens.Stream, index int) bool {
	next := docNextTokenIndex(s, index)
	if next < 0 || isPunctVal(s, next, "}") {
		return false
	}
	docContent := s.At(index).Value
	if docIsStructuralElement(s, next) {
		return true
	}
	if docIsValidControl(s, docContent, next) {
		return true
	}
	if docIsValidVariable(s, next) {
		return true
	}
	if docIsValidVariableAssignment(s, docContent, next) {
		return true
	}
	if kwIs(s.At(next), "use") {
		return true
	}
	return false
}

func docIsStructuralElement(s *tokens.Stream, index int) bool {
	t := s.At(index)
	if t.Kind == token.Keyword {
		lv := strings.ToLower(t.Value)
		if classyKeywords[lv] || structuralSkipKeywords[lv] {
			return true
		}
		if lv == "case" {
			p := prevEnumOrSwitch(s, index)
			return p >= 0 && kwIs(s.At(p), "enum")
		}
		if lv == "static" {
			n := sigNext(s, index)
			return n < 0 || !isPunctVal(s, n, "::")
		}
		return false
	}
	if t.Kind == token.Ident {
		c := strings.ToLower(t.Value)
		return c == "get" || c == "set"
	}
	return false
}

func prevEnumOrSwitch(s *tokens.Stream, index int) int {
	for j := index - 1; j >= 0; j-- {
		if s.At(j).Kind == token.Keyword {
			switch strings.ToLower(s.At(j).Value) {
			case "enum", "switch":
				return j
			}
		}
	}
	return -1
}

func docIsValidControl(s *tokens.Stream, docContent string, controlIndex int) bool {
	if s.At(controlIndex).Kind != token.Keyword {
		return false
	}
	switch strings.ToLower(s.At(controlIndex).Value) {
	case "for", "foreach", "if", "switch", "while":
	default:
		return false
	}
	open := sigNext(s, controlIndex)
	if open < 0 || !isPunctVal(s, open, "(") {
		return false
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return false
	}
	for i := open + 1; i < closeIdx; i++ {
		if s.At(i).Kind == token.Variable && strings.Contains(docContent, s.At(i).Value) {
			return true
		}
	}
	return false
}

func docIsValidVariable(s *tokens.Stream, index int) bool {
	if s.At(index).Kind != token.Variable {
		return false
	}
	n := sigNext(s, index)
	return n >= 0 && s.At(n).Kind == token.Punct && assignmentOps[s.At(n).Value]
}

func docIsValidVariableAssignment(s *tokens.Stream, docContent string, lcIndex int) bool {
	var endIdx int
	t := s.At(lcIndex)
	if t.Kind == token.Keyword && (strings.EqualFold(t.Value, "list") || strings.EqualFold(t.Value, "print") || strings.EqualFold(t.Value, "echo")) {
		endIdx = nextPunctOfKind(s, lcIndex, ")")
	} else if isPunctVal(s, lcIndex, "[") {
		endIdx = s.MatchForward(lcIndex)
	} else {
		return false
	}
	if endIdx < 0 {
		return false
	}
	for i := lcIndex + 1; i < endIdx; i++ {
		if s.At(i).Kind == token.Variable && strings.Contains(docContent, s.At(i).Value) {
			return true
		}
	}
	return false
}
