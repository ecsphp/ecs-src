package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// convertLongArray rewrites a long "name(...)" construct to "[...]", where name
// is "array" or "list". Method/constant uses (after -> ?-> ::) are skipped.
func convertLongArray(s *tokens.Stream, name string) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword && t.Kind != token.Ident {
			continue
		}
		if strings.ToLower(t.Value) != name {
			continue
		}
		if prev, ok := prevSignificant(s, i); ok {
			switch prev.Value {
			case "->", "?->", "::":
				continue
			}
		}
		j := skipWhitespace(s, i+1)
		if j >= s.Len() || s.At(j).Kind != token.Punct || s.At(j).Value != "(" {
			continue
		}
		closeIdx := s.MatchForward(j)
		if closeIdx < 0 {
			continue
		}
		s.SetValue(closeIdx, "]")
		s.SetValue(j, "[")
		for k := j - 1; k >= i; k-- {
			s.RemoveAt(k)
		}
		changed = true
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/ArraySyntaxFixer.php
//
// ArraySyntax rewrites long "array(...)" to the short "[...]" form.
type ArraySyntax struct{}

func (ArraySyntax) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\ArraySyntaxFixer`
}

func (ArraySyntax) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/ArraySyntaxFixer.php"
}

func (ArraySyntax) Fix(s *tokens.Stream) bool { return convertLongArray(s, "array") }

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ListNotation/ListSyntaxFixer.php
//
// ListSyntax rewrites "list(...)" to the short "[...]" form.
type ListSyntax struct{}

func (ListSyntax) Name() string {
	return `PhpCsFixer\Fixer\ListNotation\ListSyntaxFixer`
}

func (ListSyntax) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ListNotation/ListSyntaxFixer.php"
}

func (ListSyntax) Fix(s *tokens.Stream) bool { return convertLongArray(s, "list") }

// enclosingIsArray reports whether the innermost open bracket at index i is "[".
func topBracketIsArray(stack []string) bool {
	return len(stack) > 0 && stack[len(stack)-1] == "["
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/NoWhitespaceBeforeCommaInArrayFixer.php
//
// NoWhitespaceBeforeCommaInArray removes single-line whitespace before a comma
// inside an array.
type NoWhitespaceBeforeCommaInArray struct{}

func (NoWhitespaceBeforeCommaInArray) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\NoWhitespaceBeforeCommaInArrayFixer`
}

func (NoWhitespaceBeforeCommaInArray) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/NoWhitespaceBeforeCommaInArrayFixer.php"
}

func (NoWhitespaceBeforeCommaInArray) Fix(s *tokens.Stream) bool {
	changed := false
	var stack []string
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				stack = append(stack, t.Value)
			case ")", "]", "}":
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			case ",":
				if topBracketIsArray(stack) && i > 0 &&
					s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) {
					s.RemoveAt(i - 1)
					i--
					changed = true
				}
			}
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/WhitespaceAfterCommaInArrayFixer.php
//
// WhitespaceAfterCommaInArray ensures a single space after a comma inside an
// array (unless a newline follows, or the comma is a trailing one before "]").
type WhitespaceAfterCommaInArray struct{}

func (WhitespaceAfterCommaInArray) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\WhitespaceAfterCommaInArrayFixer`
}

func (WhitespaceAfterCommaInArray) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/WhitespaceAfterCommaInArrayFixer.php"
}

func (WhitespaceAfterCommaInArray) Fix(s *tokens.Stream) bool {
	changed := false
	var stack []string
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[", "{":
			stack = append(stack, t.Value)
		case ")", "]", "}":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case ",":
			if !topBracketIsArray(stack) || i+1 >= s.Len() {
				continue
			}
			next := s.At(i + 1)
			if next.Kind == token.Whitespace {
				continue // already spaced or newline
			}
			if next.Kind == token.Punct && next.Value == "]" {
				continue // trailing comma
			}
			s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
			i++
			changed = true
		}
	}
	return changed
}
