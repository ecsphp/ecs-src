package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/NoWhitespaceInEmptyArrayFixer.php
//
// NoWhitespaceInEmptyArray removes the whitespace inside an empty short array
// ("[ ]" -> "[]"). Only "[]" is handled; "array()" is left alone.
type NoWhitespaceInEmptyArray struct{}

func (NoWhitespaceInEmptyArray) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\NoWhitespaceInEmptyArrayFixer`
}

func (NoWhitespaceInEmptyArray) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/NoWhitespaceInEmptyArrayFixer.php"
}

func (NoWhitespaceInEmptyArray) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i+2 < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "[" {
			continue
		}
		if s.At(i+1).Kind != token.Whitespace {
			continue
		}
		if s.At(i+2).Kind != token.Punct || s.At(i+2).Value != "]" {
			continue
		}
		s.RemoveAt(i + 1)
		changed = true
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/NormalizeIndexBraceFixer.php
//
// NormalizeIndexBrace converts the deprecated curly-brace array index access
// ("$a{0}" -> "$a[0]"). To never touch a real block "{", it only converts a
// brace pair that directly follows a value that can be indexed: a variable, a
// string literal or a closing "]".
type NormalizeIndexBrace struct{}

func (NormalizeIndexBrace) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\NormalizeIndexBraceFixer`
}

func (NormalizeIndexBrace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/NormalizeIndexBraceFixer.php"
}

func (NormalizeIndexBrace) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "{" {
			continue
		}
		// PHP-CS-Fixer converts no index brace on PHP 8.4+, where hooks exist,
		// so a hook list is left alone even with hook modifiers
		if !isIndexBraceTarget(s, i) || propertyHookBraceAt(s, i, true) {
			continue
		}
		closeIdx := s.MatchForward(i)
		if closeIdx < 0 {
			continue
		}
		s.SetValue(closeIdx, "]")
		s.SetValue(i, "[")
		changed = true
	}
	return changed
}

// isIndexBraceTarget reports whether the "{" at i is an array index brace, based
// on the preceding significant token. A variable name (not the "$" of the "${}"
// variable-variable form), a string literal or a "]" mean an index access.
func isIndexBraceTarget(s *tokens.Stream, i int) bool {
	prev, ok := prevSignificant(s, i)
	if !ok {
		return false
	}
	switch {
	case prev.Kind == token.Variable && prev.Value != "$":
		return true
	case prev.Kind == token.String:
		return true
	case prev.Kind == token.Punct && prev.Value == "]":
		return true
	}
	return false
}

// isPropertyHookBrace reports whether the "{" at i opens a PHP 8.4 property
// hook list, as PHP-CS-Fixer's BraceTransformer detects it: a "get" or "set"
// hook followed by "(", "{", ";" or "=>".
func isPropertyHookBrace(s *tokens.Stream, i int) bool { return propertyHookBraceAt(s, i, false) }

// propertyHookBraceAt is isPropertyHookBrace; withModifiers also accepts the
// "final" and "&" hook modifiers, which BraceTransformer does not.
func propertyHookBraceAt(s *tokens.Stream, i int, withModifiers bool) bool {
	j := nextMeaningfulIndex(s, i)
	for j >= 0 {
		if s.At(j).Kind == token.AttributeStart {
			if j = s.AttributeEnd(j); j < 0 {
				return false
			}
		} else if !withModifiers || (!kwIs(s.At(j), "final") && !isPunctVal(s, j, "&")) {
			break
		}
		j = nextMeaningfulIndex(s, j)
	}
	if j < 0 || s.At(j).Kind != token.Ident || (!strings.EqualFold(s.At(j).Value, "get") && !strings.EqualFold(s.At(j).Value, "set")) {
		return false
	}
	j = nextMeaningfulIndex(s, j)
	if j < 0 || s.At(j).Kind != token.Punct {
		return false
	}
	switch s.At(j).Value {
	case "{", ";", "=>":
		return true
	case "(":
		closing := s.MatchForward(j)
		if closing < 0 {
			return false
		}
		after := nextMeaningfulIndex(s, closing)
		return after >= 0 && (s.At(after).Value == "{" || s.At(after).Value == "=>")
	}
	return false
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/NoMultilineWhitespaceAroundDoubleArrowFixer.php
//
// NoMultilineWhitespaceAroundDoubleArrow collapses newline-containing whitespace
// directly around "=>" to a single space. Single-line spacing is left alone, and
// a line comment before, or any comment after, the arrow is respected.
type NoMultilineWhitespaceAroundDoubleArrow struct{}

func (NoMultilineWhitespaceAroundDoubleArrow) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\NoMultilineWhitespaceAroundDoubleArrowFixer`
}

func (NoMultilineWhitespaceAroundDoubleArrow) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/NoMultilineWhitespaceAroundDoubleArrowFixer.php"
}

func (NoMultilineWhitespaceAroundDoubleArrow) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "=>" {
			continue
		}
		// before: skip when a line comment precedes the whitespace
		if i-2 < 0 || !isLineComment(s.At(i-2)) {
			changed = collapseArrowWhitespace(s, i-1) || changed
		}
		// after: skip when any comment follows the whitespace
		if i+2 >= s.Len() || !isComment(s.At(i+2)) {
			changed = collapseArrowWhitespace(s, i+1) || changed
		}
	}
	return changed
}

// collapseArrowWhitespace turns a newline-containing whitespace token into a
// single space, matching PHP-CS-Fixer's rtrim(...)." " on the arrow neighbour.
func collapseArrowWhitespace(s *tokens.Stream, i int) bool {
	if i < 0 || i >= s.Len() {
		return false
	}
	t := s.At(i)
	if t.Kind != token.Whitespace || !hasNewline(t.Value) {
		return false
	}
	s.SetValue(i, " ")
	return true
}

func isComment(t token.Token) bool {
	return t.Kind == token.Comment || t.Kind == token.DocComment
}

// commentAt reports whether index i is in range and holds a comment token.
func commentAt(s *tokens.Stream, i int) bool {
	return i >= 0 && i < s.Len() && isComment(s.At(i))
}

// isBlockOrDocComment reports whether t is a "/* */" block comment or a "/** */"
// doc comment (not a "//"/"#" line comment and not a "#[...]" attribute).
func isBlockOrDocComment(t token.Token) bool {
	if t.Kind == token.DocComment {
		return true
	}
	return t.Kind == token.Comment && strings.HasPrefix(t.Value, "/*")
}

func isLineComment(t token.Token) bool {
	if t.Kind != token.Comment {
		return false
	}
	// "#[...]" is an attribute, not a line comment
	return strings.HasPrefix(t.Value, "//") ||
		(strings.HasPrefix(t.Value, "#") && !strings.HasPrefix(t.Value, "#["))
}
