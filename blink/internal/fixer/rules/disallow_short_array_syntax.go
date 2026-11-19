package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Arrays/DisallowShortArraySyntaxSniff.php
//
// DisallowShortArraySyntax rewrites "[...]" to "array(...)". Like PHP_CodeSniffer,
// short list destructuring ("[$a, $b] = $c") counts as a short array too.
type DisallowShortArraySyntax struct{}

func (DisallowShortArraySyntax) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Arrays\DisallowShortArraySyntaxSniff`
}

func (DisallowShortArraySyntax) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Arrays/DisallowShortArraySyntaxSniff.php"
}

func (DisallowShortArraySyntax) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		if t := s.At(i); t.Kind != token.Punct || t.Value != "[" {
			continue
		}
		closeIdx := s.MatchForward(i)
		if closeIdx < 0 || !phpcsShortArrayOpen(s, i) {
			continue
		}
		s.SetValue(i, "array(")
		s.SetValue(closeIdx, ")")
		changed = true
	}
	return changed
}

// phpcsShortArrayOpen ports PHP_CodeSniffer's tokenizer test for T_OPEN_SHORT_ARRAY:
// "[" is an index unless the token before it cannot be dereferenced.
func phpcsShortArrayOpen(s *tokens.Stream, i int) bool {
	x := i - 1
	for x >= 0 && isPhpcsEmpty(s.At(x)) {
		x--
	}
	if x < 0 {
		return true
	}
	t := s.At(x)
	switch t.Kind {
	case token.Variable:
		return false
	case token.Ident:
		lower := strings.ToLower(t.Value)
		return identKeywords[lower] || lower == "true" || lower == "false" || lower == "null"
	case token.String:
		// a quoted string can be dereferenced; heredoc, nowdoc and backtick ends cannot
		v := t.Value
		return !strings.HasSuffix(v, `'`) && !strings.HasSuffix(v, `"`)
	case token.Punct:
		switch t.Value {
		case "]", "->", "?->":
			return false
		case ")":
			return parenOwnedByScope(s, x)
		case "}":
			return closesScope(s, x)
		}
	}
	return true
}

// parenOwnedByScope reports whether the ")" at i closes a control structure's
// condition ("if (...)", "foreach (...)"), so what follows starts a statement.
func parenOwnedByScope(s *tokens.Stream, i int) bool {
	open := s.MatchBackward(i)
	if open < 0 {
		return false
	}
	owner, ok := prevSignificant(s, open)
	if !ok || owner.Kind != token.Keyword {
		return false
	}
	switch strings.ToLower(owner.Value) {
	case "if", "elseif", "for", "foreach", "while", "switch", "declare", "catch":
		return true
	}
	return false
}

// closesScope reports whether the "}" at i ends a scope block (function, class,
// control structure), not a dereferencable "{...}" ("$a->{...}", "${...}") or an
// anonymous class.
func closesScope(s *tokens.Stream, i int) bool {
	open := s.MatchBackward(i)
	if open < 0 {
		return false
	}
	prev, ok := prevSignificant(s, open)
	if !ok {
		return true
	}
	switch prev.Value {
	case "->", "?->", "::", "$":
		return false
	}
	return !isAnonClassBody(s, open)
}

// isAnonClassBody reports whether the "{" at open opens a "new class" body.
func isAnonClassBody(s *tokens.Stream, open int) bool {
	for j := open - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind == token.Keyword && strings.EqualFold(t.Value, "class") {
			prev, ok := prevSignificant(s, j)
			return ok && strings.EqualFold(prev.Value, "new")
		}
		if t.Value == ";" || t.Value == "{" || t.Value == "}" {
			return false
		}
	}
	return false
}
