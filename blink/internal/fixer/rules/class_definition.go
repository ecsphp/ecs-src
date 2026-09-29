package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ClassDefinitionFixer.php
//
// ClassDefinition normalizes spacing in a class/interface/trait/enum header:
// exactly one space after the class-like keyword and around "extends" and
// "implements". Newlines (multiline implements lists) are left untouched, and
// the brace is left to braces_position.
type ClassDefinition struct{}

func (ClassDefinition) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\ClassDefinitionFixer`
}

func (ClassDefinition) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ClassDefinitionFixer.php"
}

func (ClassDefinition) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || !classLikeKeywords[strings.ToLower(t.Value)] {
			continue
		}
		if memberPrev(s, i) {
			continue // ::class
		}
		if prev, ok := prevSignificant(s, i); ok && strings.ToLower(prev.Value) == "new" {
			continue // anonymous class
		}
		if normalizeHeaderSpacing(s, i) {
			changed = true
		}
	}
	return changed
}

// normalizeHeaderSpacing collapses space runs to one after the class-like
// keyword at kw and around extends/implements, up to the opening "{".
func normalizeHeaderSpacing(s *tokens.Stream, kw int) bool {
	changed := collapseSpace(s, kw+1)
	for j := kw + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind == token.Punct && t.Value == "{" {
			break
		}
		if t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "extends", "implements":
				if collapseSpace(s, j-1) {
					changed = true
				}
				if collapseSpace(s, j+1) {
					changed = true
				}
			}
		}
	}
	return changed
}

// collapseSpace turns a single-line whitespace token at i into exactly one
// space. Multiline whitespace and non-whitespace are left alone.
func collapseSpace(s *tokens.Stream, i int) bool {
	if i < 0 || i >= s.Len() {
		return false
	}
	t := s.At(i)
	if t.Kind != token.Whitespace || hasNewline(t.Value) || t.Value == " " {
		return false
	}
	s.SetValue(i, " ")
	return true
}
