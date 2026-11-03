package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// langConstructTriggers are the language constructs the sniff requires to be
// followed by a single space. The lexer marks these words Keyword only in
// construct position; a member access ("$o->return") or a qualified name
// ("new\Foo") lexes them Ident, so a Keyword check alone excludes both.
var langConstructTriggers = map[string]bool{
	"echo": true, "print": true, "return": true, "include": true,
	"include_once": true, "require": true, "require_once": true, "new": true,
	"yield": true, "throw": true, "namespace": true, "use": true, "goto": true,
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/LanguageConstructSpacingSniff.php
//
// LanguageConstructSpacing ensures a single space between a language construct
// and its content ("echo\"x\"" -> "echo \"x\"", "return   $a" -> "return $a").
// Any whitespace run - including newlines - collapses to one space; a construct
// glued to "(" or ";" is left alone (grouping parens / empty statement).
type LanguageConstructSpacing struct{}

func (LanguageConstructSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\WhiteSpace\LanguageConstructSpacingSniff`
}

func (LanguageConstructSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/WhiteSpace/LanguageConstructSpacingSniff.php"
}

func (LanguageConstructSpacing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || !(langConstructTriggers[strings.ToLower(t.Value)] || isYieldFrom(t)) {
			continue
		}
		// "yield from" is one token: collapse the gap between the words to one
		// space, unless it holds a comment, which the sniff cannot auto-fix
		if v := t.Value; isYieldFrom(t) && !strings.ContainsAny(v, "/#") {
			if fixed := v[:len("yield")] + " " + v[len(v)-len("from"):]; fixed != v {
				s.SetValue(i, fixed)
				changed = true
			}
		}
		if langConstructSpaceAfter(s, i) {
			changed = true
		}
	}
	return changed
}

// langConstructSpaceAfter forces a single space after the construct token at idx.
// A whitespace run collapses to one space; glued content gets a space inserted,
// unless it is "(" (grouping) or ";" (no content).
func langConstructSpaceAfter(s *tokens.Stream, idx int) bool {
	if idx+1 >= s.Len() {
		return false
	}
	next := s.At(idx + 1)
	if next.Kind == token.Whitespace {
		if next.Value != " " {
			s.SetValue(idx+1, " ")
			return true
		}
		return false
	}
	if next.Kind == token.Punct && (next.Value == "(" || next.Value == ";") {
		return false
	}
	s.InsertAt(idx+1, token.Token{Kind: token.Whitespace, Value: " "})
	return true
}
