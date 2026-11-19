package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/ArbitraryParenthesesSpacingSniff.php
//
// ArbitraryParenthesesSpacing puts Spacing spaces inside parentheses that do
// not belong to a call, declaration or control structure: "( $a + 1 )" ->
// "($a + 1)".
type ArbitraryParenthesesSpacing struct {
	Spacing        int
	IgnoreNewlines bool
}

func (ArbitraryParenthesesSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\WhiteSpace\ArbitraryParenthesesSpacingSniff`
}

func (ArbitraryParenthesesSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/ArbitraryParenthesesSpacingSniff.php"
}

func (f ArbitraryParenthesesSpacing) WithConfig(config map[string]any) fixer.Fixer {
	f.Spacing = configInt(config, "spacing", f.Spacing)
	if v, ok := config["ignoreNewlines"].(bool); ok {
		f.IgnoreNewlines = v
	}
	return f
}

// arbitraryParenOwners are the words a "(" right after belongs to: a
// parenthesis owner or a function-like construct.
var arbitraryParenOwners = map[string]bool{
	"array": true, "list": true, "function": true, "fn": true, "use": true, "class": true,
	"while": true, "for": true, "foreach": true, "switch": true, "if": true, "elseif": true,
	"catch": true, "declare": true, "match": true, "isset": true, "empty": true, "unset": true,
	"eval": true, "exit": true, "die": true, "include": true, "include_once": true,
	"require": true, "require_once": true, "self": true, "parent": true, "static": true,
	"throw": true, "yield": true, "yield from": true, "from": true, "clone": true,
}

func (f ArbitraryParenthesesSpacing) Fix(s *tokens.Stream) bool {
	changed := false
	for open := 0; open < s.Len(); open++ {
		if t := s.At(open); t.Kind != token.Punct || t.Value != "(" {
			continue
		}
		if _, _, ok := phpCastAt(s, open); ok {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 || !arbitraryParen(s, open) {
			continue
		}
		next := open + 1
		for next < closeIdx && s.At(next).Kind == token.Whitespace {
			next++
		}
		if next == closeIdx {
			continue
		}
		// fix the closer first so the opener's index stays valid
		if f.fixInside(s, closeIdx-1, closeIdx) {
			changed = true
		}
		if f.fixInside(s, open+1, open) {
			changed = true
		}
	}
	return changed
}

// arbitraryParen reports whether the "(" at open is not owned by a function,
// control structure or call, mirroring the sniff's IGNORE_TOKENS check.
func arbitraryParen(s *tokens.Stream, open int) bool {
	p := phpcsPrevNonEmpty(s, open-1)
	if p < 0 {
		return true
	}
	t := s.At(p)
	switch t.Kind {
	case token.Ident, token.Variable:
		return false
	case token.Keyword:
		if arbitraryParenOwners[strings.ToLower(t.Value)] {
			return false
		}
		pp := phpcsPrevNonEmpty(s, p-1)
		return pp < 0 || !isPunctIn(s.At(pp), "->", "?->", "::", "function")
	case token.AttributeEnd:
		return true
	case token.Punct:
		switch t.Value {
		case ")", "]":
			return false
		case "&":
			pp := phpcsPrevNonEmpty(s, p-1)
			return pp < 0 || s.At(pp).Kind != token.Keyword || !isFunctionKeyword(s.At(pp).Value)
		case "}":
			o := s.MatchBackward(p)
			if o < 0 {
				return false
			}
			po := phpcsPrevNonEmpty(s, o-1)
			return po < 0 || !isPunctIn(s.At(po), "->", "?->", "::", "$")
		}
	}
	return true
}

func isPunctIn(t token.Token, vals ...string) bool {
	if t.Kind != token.Punct && t.Kind != token.Keyword {
		return false
	}
	for _, v := range vals {
		if strings.EqualFold(t.Value, v) {
			return true
		}
	}
	return false
}

func isFunctionKeyword(v string) bool {
	return strings.EqualFold(v, "function") || strings.EqualFold(v, "fn")
}

// fixInside fixes the spacing at the whitespace position ws, inside the
// parenthesis at paren.
func (f ArbitraryParenthesesSpacing) fixInside(s *tokens.Stream, ws, paren int) bool {
	expected := strings.Repeat(" ", max(f.Spacing, 0))
	if !phpcsWhitespaceAt(s, ws) {
		if expected == "" {
			return false
		}
		s.InsertAt(max(ws, paren), token.Token{Kind: token.Whitespace, Value: expected})
		return true
	}
	v := s.At(ws).Value[phpcsCommentNewline(s, ws):]
	if strings.ContainsAny(v, "\r\n") {
		if f.IgnoreNewlines {
			return false
		}
	} else if phpcsTabWidth(s, ws) == f.Spacing {
		return false
	}
	if expected == "" {
		dropPhpcsWhitespace(s, ws)
		return true
	}
	keep := s.At(ws).Value[:phpcsCommentNewline(s, ws)]
	if keep+expected == s.At(ws).Value {
		return false
	}
	s.SetValue(ws, keep+expected)
	return true
}
