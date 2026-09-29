package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoAlternativeSyntaxFixer.php
//
// NoAlternativeSyntax replaces control-structure alternative syntax with braces
// ("if (x): ... endif;" -> "if (x) { ... }"). Matches ECS's default
// (fix_non_monolithic_code=true), so inline-HTML blocks are converted too.
type NoAlternativeSyntax struct{}

func (NoAlternativeSyntax) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\NoAlternativeSyntaxFixer`
}

func (NoAlternativeSyntax) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoAlternativeSyntaxFixer.php"
}

var nasOpenControls = map[string]bool{
	"if": true, "foreach": true, "while": true, "for": true, "switch": true, "declare": true,
}

var nasEndControls = map[string]bool{
	"endif": true, "endforeach": true, "endwhile": true, "endfor": true,
	"endswitch": true, "enddeclare": true,
}

func nasIsKeyword(t token.Token, set map[string]bool) bool {
	return t.Kind == token.Keyword && set[strings.ToLower(t.Value)]
}

func (NoAlternativeSyntax) Fix(s *tokens.Stream) bool {
	changed := false
	for index := s.Len() - 1; index >= 0; index-- {
		tok := s.At(index)
		if tok.Kind == token.Keyword && strings.EqualFold(tok.Value, "elseif") {
			changed = nasFixElseif(s, index) || changed
			continue
		}
		if tok.Kind == token.Keyword && strings.EqualFold(tok.Value, "else") {
			changed = nasFixElse(s, index) || changed
			continue
		}
		changed = nasFixOpenClose(s, index) || changed
	}
	return changed
}

func nasFixOpenClose(s *tokens.Stream, index int) bool {
	tok := s.At(index)
	if nasIsKeyword(tok, nasOpenControls) {
		openIndex := nasNextParen(s, index)
		if openIndex == -1 {
			return false
		}
		closeIndex := s.MatchForward(openIndex)
		if closeIndex == -1 {
			return false
		}
		afterIndex := nextMeaningfulIndex(s, closeIndex)
		if afterIndex == -1 || !nasIsPunct(s.At(afterIndex), ":") {
			return false
		}
		var items []token.Token
		if afterIndex-1 >= 0 && s.At(afterIndex-1).Kind != token.Whitespace {
			items = append(items, token.Token{Kind: token.Whitespace, Value: " "})
		}
		items = append(items, token.Token{Kind: token.Punct, Value: "{"})
		if afterIndex+1 < s.Len() && s.At(afterIndex+1).Kind != token.Whitespace {
			items = append(items, token.Token{Kind: token.Whitespace, Value: " "})
		}
		nasReplace(s, afterIndex, items)
		return true
	}

	if !nasIsKeyword(tok, nasEndControls) {
		return false
	}
	nextIndex := nextMeaningfulIndex(s, index)
	s.Set(index, token.Token{Kind: token.Punct, Value: "}"})
	if nextIndex != -1 && nasIsPunct(s.At(nextIndex), ";") {
		s.RemoveAt(nextIndex)
	}
	return true
}

func nasFixElse(s *tokens.Stream, index int) bool {
	afterIndex := nextMeaningfulIndex(s, index)
	if afterIndex == -1 || !nasIsPunct(s.At(afterIndex), ":") {
		return false
	}
	nasAddBraces(s, token.Token{Kind: token.Keyword, Value: "else"}, index, afterIndex)
	return true
}

func nasFixElseif(s *tokens.Stream, index int) bool {
	parenEnd := nasFindParenthesisEnd(s, index)
	afterIndex := nextMeaningfulIndex(s, parenEnd)
	if afterIndex == -1 || !nasIsPunct(s.At(afterIndex), ":") {
		return false
	}
	nasAddBraces(s, token.Token{Kind: token.Keyword, Value: "elseif"}, index, afterIndex)
	return true
}

// nasAddBraces turns "else :" / "elseif (...) :" into "} else {" / "} elseif (...) {".
func nasAddBraces(s *tokens.Stream, keyword token.Token, index, colonIndex int) {
	trailingWs := index+1 < s.Len() && s.At(index+1).Kind != token.Whitespace
	open := []token.Token{
		{Kind: token.Punct, Value: "}"},
		{Kind: token.Whitespace, Value: " "},
		keyword,
	}
	if trailingWs {
		open = append(open, token.Token{Kind: token.Whitespace, Value: " "})
	}
	nasReplace(s, index, open)

	colonIndex += len(open) - 1
	closing := []token.Token{{Kind: token.Punct, Value: "{"}}
	if colonIndex+1 < s.Len() && s.At(colonIndex+1).Kind != token.Whitespace {
		closing = append(closing, token.Token{Kind: token.Whitespace, Value: " "})
	}
	nasReplace(s, colonIndex, closing)
}

// nasReplace replaces the single token at pos with items, preserving order.
func nasReplace(s *tokens.Stream, pos int, items []token.Token) {
	s.RemoveAt(pos)
	for i, t := range items {
		s.InsertAt(pos+i, t)
	}
}

// nasNextParen returns the index of the first "(" after index, or -1.
func nasNextParen(s *tokens.Stream, index int) int {
	for j := index + 1; j < s.Len(); j++ {
		if nasIsPunct(s.At(j), "(") {
			return j
		}
	}
	return -1
}

func nasIsPunct(t token.Token, v string) bool {
	return t.Kind == token.Punct && t.Value == v
}

// nasFindParenthesisEnd returns the ")" index of the control's condition, or the
// control token index itself when it has no parenthesis.
func nasFindParenthesisEnd(s *tokens.Stream, controlIndex int) int {
	ni := nextMeaningfulIndex(s, controlIndex)
	if ni == -1 || !nasIsPunct(s.At(ni), "(") {
		return controlIndex
	}
	end := s.MatchForward(ni)
	if end == -1 {
		return controlIndex
	}
	return end
}
