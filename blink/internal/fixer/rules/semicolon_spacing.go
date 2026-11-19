package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/SemicolonSpacingSniff.php
//
// SemicolonSpacing moves a ";" separated by whitespace or comments onto the
// code it ends: "$a /* c */ ;" -> "$a; /* c */". Empty statements are left alone.
type SemicolonSpacing struct{}

func (SemicolonSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\WhiteSpace\SemicolonSpacingSniff`
}

func (SemicolonSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/WhiteSpace/SemicolonSpacingSniff.php"
}

func (SemicolonSpacing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 1; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Punct || t.Value != ";" || !isPhpcsEmpty(s.At(i-1)) {
			continue
		}
		forOpen := forConditionParen(s, i)
		nonSpace := i - 1
		for nonSpace >= 0 && (s.At(nonSpace).Kind == token.Whitespace || (forOpen < 0 && isPhpcsEmpty(s.At(nonSpace)))) {
			nonSpace--
		}
		if nonSpace < 0 || s.At(nonSpace).Value == ";" || nonSpace == forOpen || isPhpcsScopeOpener(s, nonSpace) {
			continue
		}
		semi := s.At(i)
		s.RemoveAt(i)
		if w := i - 1; s.At(w).Kind == token.Whitespace {
			dropPhpcsWhitespace(s, w)
		}
		s.InsertAt(nonSpace+1, semi)
		changed = true
	}
	return changed
}

// forConditionParen returns the "(" of the "for (...)" whose parentheses
// directly hold the token at i, or -1.
func forConditionParen(s *tokens.Stream, i int) int {
	for j := i - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case ")", "]", "}":
			if open := s.MatchBackward(j); open >= 0 {
				j = open
			}
		case "(":
			if owner, ok := prevSignificant(s, j); ok && owner.Kind == token.Keyword && strings.EqualFold(owner.Value, "for") {
				return j
			}
			return -1
		}
	}
	return -1
}

// isPhpcsScopeOpener reports whether the token at i opens a scope for
// PHP_CodeSniffer: a "{" owned by a declaration or control structure, or the
// ":" of alternative syntax or a case.
func isPhpcsScopeOpener(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind != token.Punct {
		return false
	}
	switch t.Value {
	case "{":
		prev, ok := prevSignificant(s, i)
		if !ok {
			return false
		}
		switch prev.Kind {
		case token.OpenTag, token.Variable:
			return false
		case token.Punct:
			switch prev.Value {
			case ";", "{", "}", ":", "->", "?->", "::", "$":
				return false
			}
		}
		return true
	case ":":
		prev, ok := prevSignificant(s, i)
		if !ok {
			return false
		}
		if prev.Kind == token.Keyword && strings.EqualFold(prev.Value, "else") {
			return true
		}
		if prev.Value == ")" {
			return parenOwnedByScope(s, prevSignificantIndex(s, i))
		}
		return isCaseColon(s, i)
	}
	return false
}

// isCaseColon reports whether the ":" at i ends a "case ..." or "default" label.
func isCaseColon(s *tokens.Stream, i int) bool {
	for j := i - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "case", "default":
				return true
			}
		}
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case ")", "]", "}":
			if open := s.MatchBackward(j); open >= 0 && t.Value != "}" {
				j = open
				continue
			}
			return false
		case ";", "{", "?", ":":
			return false
		}
	}
	return false
}
