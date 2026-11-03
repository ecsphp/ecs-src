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
		if t.Kind != token.Keyword || !langConstructTriggers[strings.ToLower(t.Value)] {
			continue
		}
		// "yield from": collapse the gap to one space, then space the content
		// after "from"; a comment between the words is left untouched, as the
		// sniff cannot auto-fix it.
		if strings.EqualFold(t.Value, "yield") {
			if n := sigNext(s, i); n >= 0 && s.At(n).Kind == token.Ident && strings.EqualFold(s.At(n).Value, "from") {
				if n == i+2 && s.At(i+1).Kind == token.Whitespace && s.At(i+1).Value != " " {
					s.SetValue(i+1, " ")
					changed = true
				}
				if langConstructSpaceAfter(s, n) {
					changed = true
				}
				i = n
				continue
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
