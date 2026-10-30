package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ClassDefinitionFixer.php
//
// ClassDefinition normalizes spacing in a class/interface/trait/enum header:
// exactly one space after the class-like keyword and around "extends" and
// "implements". Newlines (multiline implements lists) are left untouched, and
// the brace is left to braces_position.
//
// The fields mirror the fixer options; their zero values reproduce today's
// behavior (multiline headers kept, anonymous classes untouched):
//   - singleLine: "single_line" collapses a multiline header onto one line
//   - singleItemSingleLine: "single_item_single_line" collapses only when the
//     extends/implements clause holds a single item
//   - spaceBeforeParenthesis: "space_before_parenthesis" adds a space before the
//     constructor "(" of an anonymous class
//   - multiLineExtendsEachSingleLine / inlineConstructorArguments are accepted
//     but not yet implemented (see the package report)
type ClassDefinition struct {
	singleLine                     bool
	singleItemSingleLine           bool
	multiLineExtendsEachSingleLine bool
	spaceBeforeParenthesis         bool
	inlineConstructorArguments     bool
}

func (f ClassDefinition) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["single_line"].(bool); ok {
		f.singleLine = v
	}
	if v, ok := config["single_item_single_line"].(bool); ok {
		f.singleItemSingleLine = v
	}
	if v, ok := config["multi_line_extends_each_single_line"].(bool); ok {
		f.multiLineExtendsEachSingleLine = v
	}
	if v, ok := config["space_before_parenthesis"].(bool); ok {
		f.spaceBeforeParenthesis = v
	}
	if v, ok := config["inline_constructor_arguments"].(bool); ok {
		f.inlineConstructorArguments = v
	}
	return f
}

func (ClassDefinition) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\ClassDefinitionFixer`
}

func (ClassDefinition) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ClassDefinitionFixer.php"
}

func (f ClassDefinition) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || !classLikeKeywords[strings.ToLower(t.Value)] {
			continue
		}
		if memberPrev(s, i) {
			continue // ::class
		}
		if newIdx := anonClassNewIndex(s, i); newIdx >= 0 {
			// anonymous class: the constructor "(" spacing is normalized (opt-in),
			// and a multiline attribute in the header ("new #[Attr(\n...)] class")
			// is flattened to one line, as php-cs-fixer's makeClassyDefinitionSingleLine
			// collapses the whole anon-class header
			if f.spaceBeforeParenthesis && classDefSpaceBeforeParen(s, i) {
				changed = true
			}
			if classDefFlattenAnonHeaderAttributes(s, newIdx, i) {
				changed = true
			}
			continue
		}
		// a genuine declaration is followed by the type name; a class-like keyword
		// followed by "::" or "(" (e.g. "Enum::MODE_ADD") is a name the lexer tagged
		// as a keyword, not a declaration
		if n := nextSignificantIndex(s, i); n < 0 || s.At(n).Kind != token.Ident {
			continue
		}
		if normalizeHeaderSpacing(s, i) {
			changed = true
		}
		if (f.singleLine || f.singleItemSingleLine) && f.classDefFlattenHeader(s, i) {
			changed = true
		}
	}
	return changed
}

// anonClassNewIndex returns the index of the "new" keyword governing the
// anonymous class whose keyword is at classIdx, or -1 for a named declaration.
// prevMeaningfulIndex steps over a leading attribute ("new #[Attr] class").
func anonClassNewIndex(s *tokens.Stream, classIdx int) int {
	p := prevMeaningfulSkippingAttributes(s, classIdx)
	if p >= 0 && s.At(p).Kind == token.Keyword && strings.EqualFold(s.At(p).Value, "new") {
		return p
	}
	return -1
}

// classDefFlattenAnonHeaderAttributes collapses a multiline attribute standing
// between the "new" at newIdx and the "class" keyword at classIdx back onto one
// line, mirroring makeClassyDefinitionSingleLine over the anon-class header.
func classDefFlattenAnonHeaderAttributes(s *tokens.Stream, newIdx, classIdx int) bool {
	changed := false
	for j := newIdx + 1; j < classIdx; j++ {
		if !attributeStartsAt(s, j) {
			continue
		}
		end := s.AttributeEnd(j)
		if end < 0 {
			continue
		}
		before := s.Len()
		if strings.ContainsRune(attributeText(s, j, end), '\n') &&
			rewriteAttributeText(s, j, end, func(v string) (string, bool) {
				return collapseAttributeText(v, false)
			}) {
			changed = true
			classIdx -= before - s.Len() // the splice shifted the "class" keyword
		}
		j = s.AttributeEnd(j) // skip past the attribute we just handled
	}
	return changed
}

// classDefSpaceBeforeParen ensures exactly one single-line space between the
// class-like keyword at classIdx and an immediately following "(".
func classDefSpaceBeforeParen(s *tokens.Stream, classIdx int) bool {
	n := nextSignificantIndex(s, classIdx)
	if n < 0 || s.At(n).Kind != token.Punct || s.At(n).Value != "(" {
		return false
	}
	if classIdx+1 == n {
		s.InsertAt(n, token.Token{Kind: token.Whitespace, Value: " "})
		return true
	}
	if s.At(classIdx+1).Kind == token.Whitespace && !hasNewline(s.At(classIdx+1).Value) && s.At(classIdx+1).Value != " " {
		s.SetValue(classIdx+1, " ")
		return true
	}
	return false
}

// classDefFlattenHeader collapses newline-spanning whitespace in the class header
// (between the keyword and the "{") to a single space, for single_line and
// single_item_single_line. Whitespace next to a docblock or attribute, and the
// whitespace immediately before the "{", is preserved.
func (f ClassDefinition) classDefFlattenHeader(s *tokens.Stream, kw int) bool {
	open := nextPunctOfKind(s, kw, "{")
	if open < 0 {
		return false
	}
	if !f.singleLine && f.singleItemSingleLine {
		if classDefTopLevelCommas(s, kw+1, open) != 0 {
			return false // more than one item: single_item_single_line does not apply
		}
	}
	end := prevSignificantIndex(s, open)
	if end < 0 {
		return false
	}
	changed := false
	for j := kw + 1; j <= end; j++ {
		t := s.At(j)
		if t.Kind != token.Whitespace || !hasNewline(t.Value) {
			continue
		}
		if j-1 >= 0 && (s.At(j-1).Kind == token.DocComment || attributeEndsAt(s, j-1)) {
			continue
		}
		if j+1 < s.Len() && (s.At(j+1).Kind == token.DocComment || attributeStartsAt(s, j+1)) {
			continue
		}
		s.SetValue(j, " ")
		changed = true
	}
	return changed
}

// classDefTopLevelCommas counts commas at depth zero between from and to.
func classDefTopLevelCommas(s *tokens.Stream, from, to int) int {
	depth, count := 0, 0
	for j := from; j < to && j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[":
			depth++
		case ")", "]":
			depth--
		case ",":
			if depth == 0 {
				count++
			}
		}
	}
	return count
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
