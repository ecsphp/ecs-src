package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// This file implements three PHP-CS-Fixer ControlStructure rules on the flat
// token stream. Each is a deliberately narrow subset of the upstream fixer:
// only token cases that are provably semantics-preserving and idempotent are
// rewritten, everything else is left untouched.

// includeKeywords are the four language constructs handled by IncludeFixer.
var includeKeywords = map[string]bool{
	"include": true, "include_once": true, "require": true, "require_once": true,
}

// isMemberAccessPrev reports whether the significant token before i makes the
// name at i a member access ("$o->include", "Foo::for") rather than a language
// keyword.
func isMemberAccessPrev(s *tokens.Stream, i int) bool {
	p := sigPrev(s, i)
	if p < 0 {
		return false
	}
	pt := s.At(p)
	return pt.Kind == token.Punct && (pt.Value == "->" || pt.Value == "?->" || pt.Value == "::")
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/IncludeFixer.php
//
// Include normalizes the whitespace between an include/require construct and its
// argument to exactly one space ("require  \"x\"" -> "require \"x\""). Only the
// spacing part of the upstream fixer is implemented; the paren-form
// ("include(\"x\")") is left untouched because removing those parentheses cannot
// be proven safe on a flat token stream.
type Include struct{}

func (Include) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\IncludeFixer`
}

func (Include) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/IncludeFixer.php"
}

func (Include) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || !includeKeywords[strings.ToLower(t.Value)] {
			continue
		}
		if isMemberAccessPrev(s, i) {
			continue
		}
		n := i + 1
		if n >= s.Len() {
			continue
		}
		nt := s.At(n)
		if nt.Kind == token.Whitespace {
			if hasNewline(nt.Value) {
				continue // keep multi-line spacing (alignment) untouched
			}
			j := sigNext(s, i)
			if j < 0 || (s.At(j).Kind == token.Punct && s.At(j).Value == "(") {
				continue // paren form: leave the parentheses alone
			}
			if nt.Value != " " {
				s.SetValue(n, " ")
				changed = true
			}
			continue
		}
		// Directly adjacent argument: insert a single space. Only the token kinds
		// that can legally abut the keyword are handled ("include$x", "include'x'",
		// "include\\Foo"); the paren form is skipped.
		switch {
		case nt.Kind == token.Variable || nt.Kind == token.String:
		case nt.Kind == token.Punct && nt.Value == `\`:
		default:
			continue
		}
		s.InsertAt(n, token.Token{Kind: token.Whitespace, Value: " "})
		changed = true
	}
	return changed
}

// loopBodyKeywords are the loops EmptyLoopBody acts on (for/foreach/while).
var loopBodyKeywords = map[string]bool{
	"for": true, "foreach": true, "while": true,
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/EmptyLoopBodyFixer.php
//
// EmptyLoopBody rewrites an empty braced loop body to a semicolon, matching the
// upstream default style "semicolon" ("while ($x) {}" -> "while ($x);"). A
// do-while is naturally excluded (its "while (...)" is followed by ";", not
// "{"), and a body holding a comment is kept.
type EmptyLoopBody struct{}

func (EmptyLoopBody) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\EmptyLoopBodyFixer`
}

func (EmptyLoopBody) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/EmptyLoopBodyFixer.php"
}

func (EmptyLoopBody) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || !loopBodyKeywords[strings.ToLower(t.Value)] {
			continue
		}
		if isMemberAccessPrev(s, i) {
			continue
		}
		open := sigNext(s, i)
		if open < 0 || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
			continue
		}
		closeParen := s.MatchForward(open)
		if closeParen < 0 {
			continue
		}
		braceOpen := sigNext(s, closeParen)
		if braceOpen < 0 || s.At(braceOpen).Kind != token.Punct || s.At(braceOpen).Value != "{" {
			continue
		}
		// The body is empty only when the next non-whitespace token (comments are
		// NOT skipped, so a commented body is preserved) is the closing brace.
		braceClose := nextSignificantIndex(s, braceOpen)
		if braceClose < 0 || s.At(braceClose).Kind != token.Punct || s.At(braceClose).Value != "}" {
			continue
		}
		// Replace "{" with ";" and drop the "}" plus the whitespace-only interior,
		// so "{ }" collapses cleanly to ";".
		s.SetValue(braceOpen, ";")
		s.ReplaceRange(braceOpen+1, braceClose, nil)
		changed = true
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/EmptyLoopConditionFixer.php
//
// EmptyLoopCondition rewrites an empty for-loop header to while(true), matching
// the upstream default style "while" ("for (;;)" -> "while (true)"). Only this
// for-form is implemented; the do-while conversion of the upstream fixer is not,
// and a header containing a comment is skipped to avoid reordering it.
type EmptyLoopCondition struct{}

func (EmptyLoopCondition) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\EmptyLoopConditionFixer`
}

func (EmptyLoopCondition) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/EmptyLoopConditionFixer.php"
}

func (EmptyLoopCondition) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "for" {
			continue
		}
		if isMemberAccessPrev(s, i) {
			continue
		}
		open := sigNext(s, i)
		if open < 0 || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
			continue
		}
		closeParen := s.MatchForward(open)
		if closeParen < 0 {
			continue
		}
		// Empty condition: "( ; ; )" - exactly two semicolons and nothing else.
		a := sigNext(s, open)
		if a < 0 || s.At(a).Kind != token.Punct || s.At(a).Value != ";" {
			continue
		}
		b := sigNext(s, a)
		if b < 0 || s.At(b).Kind != token.Punct || s.At(b).Value != ";" {
			continue
		}
		if c := sigNext(s, b); c != closeParen {
			continue
		}
		// Only rewrite when the header holds no comment, so nothing is dropped or
		// reordered by the replacement.
		if !spanClean(s, i, closeParen) {
			continue
		}
		s.ReplaceRange(i, closeParen, []token.Token{
			{Kind: token.Keyword, Value: "while"},
			{Kind: token.Whitespace, Value: " "},
			{Kind: token.Punct, Value: "("},
			{Kind: token.Ident, Value: "true"},
			{Kind: token.Punct, Value: ")"},
		})
		changed = true
	}
	return changed
}
