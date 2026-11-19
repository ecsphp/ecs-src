package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Functions/NullableTypeDeclarationSniff.php
//
// NullableTypeDeclarationSniff removes whitespace between a nullable "?" and its
// type: "? int" -> "?int". A comment in between is left alone.
type NullableTypeDeclarationSniff struct{}

func (NullableTypeDeclarationSniff) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Functions\NullableTypeDeclarationSniff`
}

func (NullableTypeDeclarationSniff) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Functions/NullableTypeDeclarationSniff.php"
}

func (NullableTypeDeclarationSniff) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Punct || t.Value != "?" || !phpcsIsNullable(s, i) {
			continue
		}
		next := i + 1
		for next < s.Len() && s.At(next).Kind == token.Whitespace {
			next++
		}
		if next == i+1 || next >= s.Len() || !phpcsNullableTypeStart(s.At(next)) {
			continue
		}
		for j := next - 1; j > i; j-- {
			s.RemoveAt(j)
		}
		changed = true
	}
	return changed
}

// phpcsNullableTypeStart mirrors the sniff's VALID_TOKENS: a name, or
// callable/self/parent/static/null/false/true.
func phpcsNullableTypeStart(t token.Token) bool {
	switch t.Kind {
	case token.Ident:
		return true
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "callable", "static", "array", "namespace":
			return true
		}
	case token.Punct:
		return t.Value == `\`
	}
	return false
}

// phpcsIsNullable mirrors how PHP_CodeSniffer's tokenizer tells a nullable type
// "?" (T_NULLABLE) from a ternary "?" (T_INLINE_THEN).
func phpcsIsNullable(s *tokens.Stream, q int) bool {
	if p := phpcsPrevNonEmpty(s, q-1); p >= 0 && s.At(p).Kind == token.Keyword && strings.EqualFold(s.At(p).Value, "const") {
		return true
	}
	// a type must follow: a name, "array", "namespace", "static", "\" or "callable"
	lastName := ""
	i := q + 1
	for ; i < s.Len(); i++ {
		t := s.At(i)
		if isPhpcsEmpty(t) {
			continue
		}
		if kind := phpcsNullableNamePart(t); kind != "" {
			lastName = kind
			continue
		}
		switch {
		case lastName == "" && (t.Kind != token.Keyword || !strings.EqualFold(t.Value, "callable")):
			return false
		case lastName == "array" && t.Value == "(":
			return false
		case lastName == "name" && (t.Value == "::" || t.Value == "(" || t.Value == ":"):
			return false
		}
		break
	}
	if i >= s.Len() {
		return false
	}
	lastSeen := ""
	first := true
	for j := q - 1; j >= 0; j-- {
		t := s.At(j)
		lower := strings.ToLower(t.Value)
		if t.Kind == token.Keyword && lower == "static" && (lastSeen == "::" || lastSeen == "(") {
			lastSeen = lower
			continue
		}
		if first && !isPhpcsEmpty(t) {
			if t.Kind == token.AttributeEnd || (t.Kind == token.Punct && (t.Value == ":" || t.Value == ",")) {
				return true
			}
			first = false
		}
		if t.Kind == token.Keyword && phpcsNullableModifier(lower) {
			return true
		}
		switch {
		case t.Kind == token.OpenTag:
			return false
		case t.Kind == token.Punct && (t.Value == "=>" || t.Value == "=" || t.Value == "{" || t.Value == ";"):
			return false
		}
		if !isPhpcsEmpty(t) {
			lastSeen = t.Value
		}
	}
	return false
}

func phpcsNullableNamePart(t token.Token) string {
	switch {
	case t.Kind == token.Ident:
		return "name"
	case t.Kind == token.Punct && t.Value == `\`:
		return "name"
	case t.Kind == token.Keyword && (strings.EqualFold(t.Value, "namespace") || strings.EqualFold(t.Value, "static")):
		return "name"
	case t.Kind == token.Keyword && strings.EqualFold(t.Value, "array"):
		return "array"
	}
	return ""
}

// phpcsNullableModifier: function, fn, var, readonly and the method/scope modifiers
// (including asymmetric "public(set)").
func phpcsNullableModifier(lower string) bool {
	switch lower {
	case "function", "fn", "var", "readonly", "abstract", "static", "final", "public", "protected", "private":
		return true
	}
	return strings.HasPrefix(lower, "public(") || strings.HasPrefix(lower, "protected(") || strings.HasPrefix(lower, "private(")
}

// phpcsPrevNonEmpty is the index of the last token at or before i that is not
// whitespace or a comment, -1 when there is none.
func phpcsPrevNonEmpty(s *tokens.Stream, i int) int {
	for i >= 0 && isPhpcsEmpty(s.At(i)) {
		i--
	}
	return i
}
