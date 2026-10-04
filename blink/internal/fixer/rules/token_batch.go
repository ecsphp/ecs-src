package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// nextSignificantIndex returns the index of the first non-whitespace token after
// i, or -1 if there is none.
func nextSignificantIndex(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		if s.Kind(j) != token.Whitespace {
			return j
		}
	}
	return -1
}

// stringQuote returns the quote byte of a well-formed string literal value
// (opening and matching closing quote), or 0 when the value is not one.
func stringQuote(v string) byte {
	if len(v) >= 2 && (v[0] == '\'' || v[0] == '"') && v[len(v)-1] == v[0] {
		return v[0]
	}
	return 0
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/AttributeNotation/AttributeBlockNoSpacesFixer.php
//
// AttributeBlockNoSpaces trims spaces just inside the outer brackets of a PHP 8
// attribute: "#[ Foo ]" -> "#[Foo]". Inner content is left intact.
type AttributeBlockNoSpaces struct{}

func (AttributeBlockNoSpaces) Name() string {
	return `PhpCsFixer\Fixer\AttributeNotation\AttributeBlockNoSpacesFixer`
}

func (AttributeBlockNoSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/AttributeNotation/AttributeBlockNoSpacesFixer.php"
}

func (AttributeBlockNoSpaces) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Comment || !strings.HasPrefix(t.Value, "#[") || !strings.HasSuffix(t.Value, "]") {
			continue
		}
		inner := t.Value[2 : len(t.Value)-1]
		trimmed := strings.TrimRight(strings.TrimLeft(inner, " \t"), " \t")
		if trimmed == inner {
			continue
		}
		s.SetValue(i, "#["+trimmed+"]")
		changed = true
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NoSpaceAroundDoubleColonFixer.php
//
// NoSpaceAroundDoubleColon removes single-line whitespace around a "::" operator
// ("Foo :: bar" -> "Foo::bar"). Whitespace spanning a newline is kept.
type NoSpaceAroundDoubleColon struct{}

func (NoSpaceAroundDoubleColon) Name() string {
	return `PhpCsFixer\Fixer\Operator\NoSpaceAroundDoubleColonFixer`
}

func (NoSpaceAroundDoubleColon) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NoSpaceAroundDoubleColonFixer.php"
}

func (NoSpaceAroundDoubleColon) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "::" {
			continue
		}
		if i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) {
			s.RemoveAt(i + 1)
			changed = true
		}
		if i-1 >= 0 && s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) {
			s.RemoveAt(i - 1)
			i--
			changed = true
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/TrimArraySpacesFixer.php
//
// TrimArraySpaces removes single-line whitespace just inside an array-literal
// "[" ... "]" ("[ 1, 2 ]" -> "[1, 2]"). Offset access is left alone. Long
// "array( ... )" syntax is not handled (short "[]" literals only).
type TrimArraySpaces struct{}

func (TrimArraySpaces) Name() string {
	return `PhpCsFixer\Fixer\ArrayNotation\TrimArraySpacesFixer`
}

func (TrimArraySpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ArrayNotation/TrimArraySpacesFixer.php"
}

func (TrimArraySpaces) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "[" || isOffsetOpen(s, i) {
			continue
		}
		closeIdx := s.MatchForward(i)
		if closeIdx < 0 {
			continue
		}
		// a space kept next to a comment is left alone ("[ // note" / "/* note */ ]")
		if closeIdx-1 > i && s.At(closeIdx-1).Kind == token.Whitespace && !hasNewline(s.At(closeIdx-1).Value) &&
			!commentAt(s, closeIdx-2) {
			s.RemoveAt(closeIdx - 1)
			changed = true
		}
		if i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) &&
			!commentAt(s, i+2) {
			s.RemoveAt(i + 1)
			changed = true
		}
	}
	return changed
}

// nativeTypeNames are the built-in type names lowercased in type declarations.
// Every entry is a reserved word, so none can name a user class or constant.
var nativeTypeNames = map[string]bool{
	"int": true, "string": true, "bool": true, "float": true, "void": true,
	"array": true, "iterable": true, "object": true, "mixed": true, "null": true,
	"false": true, "true": true, "never": true, "callable": true, "self": true,
	"parent": true, "static": true,
}

// typeContextPrev reports whether the significant token before i marks a type
// position (a signature/property/return-type boundary).
func typeContextPrev(s *tokens.Stream, i int) bool {
	p := prevSignificantIndex(s, i)
	if p < 0 {
		return false
	}
	pt := s.At(p)
	if pt.Kind == token.Punct {
		switch pt.Value {
		case "(", ",", "|", "?", "&", ":":
			return true
		}
		return false
	}
	if pt.Kind == token.Keyword {
		switch strings.ToLower(pt.Value) {
		case "public", "private", "protected", "static", "readonly", "var":
			return true
		}
	}
	return false
}

// typeContextNext reports whether the significant token after i is consistent
// with a type declaration (a variable, a union/intersection, or a body/end).
func typeContextNext(s *tokens.Stream, i int) bool {
	n := nextSignificantIndex(s, i)
	if n < 0 {
		return false
	}
	nt := s.At(n)
	if nt.Kind == token.Variable {
		return true
	}
	if nt.Kind == token.Punct {
		switch nt.Value {
		case "|", "&", "{", ";":
			return true
		}
	}
	return false
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/NativeTypeDeclarationCasingFixer.php
//
// NativeTypeDeclarationCasing lowercases native type names used in type
// declarations (parameters, properties, return types). It only ever touches
// reserved words, which cannot name a class, and only in a type position.
type NativeTypeDeclarationCasing struct{}

func (NativeTypeDeclarationCasing) Name() string {
	return `PhpCsFixer\Fixer\Casing\NativeTypeDeclarationCasingFixer`
}

func (NativeTypeDeclarationCasing) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/NativeTypeDeclarationCasingFixer.php"
}

func (NativeTypeDeclarationCasing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Ident && t.Kind != token.Keyword {
			continue
		}
		lower := strings.ToLower(t.Value)
		if !nativeTypeNames[lower] || t.Value == lower {
			continue
		}
		if !typeContextPrev(s, i) || !typeContextNext(s, i) {
			continue
		}
		s.SetValue(i, lower)
		changed = true
	}
	return changed
}

// isLabelByte reports whether c is valid in a heredoc/nowdoc label.
func isLabelByte(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// heredocToNowdoc converts a heredoc token value to a nowdoc by quoting the
// label, when the label is unquoted and the body has no interpolation ("$" or
// "{") and no backslash escapes. It returns the new value and whether it changed.
func heredocToNowdoc(v string) (string, bool) {
	if !strings.HasPrefix(v, "<<<") {
		return v, false
	}
	i := 3
	for i < len(v) && (v[i] == ' ' || v[i] == '\t') {
		i++
	}
	if i < len(v) && (v[i] == '\'' || v[i] == '"') {
		return v, false // already quoted (nowdoc or explicit heredoc)
	}
	labelStart := i
	for i < len(v) && isLabelByte(v[i]) {
		i++
	}
	if i == labelStart {
		return v, false
	}
	labelEnd := i
	_, after, ok := strings.Cut(v, "\n")
	if !ok {
		return v, false
	}
	body := after
	if strings.ContainsAny(body, "$\\") || strings.Contains(body, "{") {
		return v, false
	}
	return v[:labelStart] + "'" + v[labelStart:labelEnd] + "'" + v[labelEnd:], true
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/HeredocToNowdocFixer.php
//
// HeredocToNowdoc converts a heredoc with no interpolation or escapes into a
// nowdoc by quoting its label ("<<<EOT" -> "<<<'EOT'").
type HeredocToNowdoc struct{}

func (HeredocToNowdoc) Name() string {
	return `PhpCsFixer\Fixer\StringNotation\HeredocToNowdocFixer`
}

func (HeredocToNowdoc) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/HeredocToNowdocFixer.php"
}

func (HeredocToNowdoc) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.String || !strings.HasPrefix(t.Value, "<<<") {
			continue
		}
		if out, ok := heredocToNowdoc(t.Value); ok {
			s.SetValue(i, out)
			changed = true
		}
	}
	return changed
}

// rangeHasNewline reports whether any whitespace token strictly between lo and
// hi spans a newline.
func rangeHasNewline(s *tokens.Stream, lo, hi int) bool {
	for j := lo + 1; j < hi; j++ {
		if s.At(j).Kind == token.Whitespace && hasNewline(s.At(j).Value) {
			return true
		}
	}
	return false
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NoUselessConcatOperatorFixer.php
//
// NoUselessConcatOperator merges two adjacent string literals joined by "." into
// one when they share a quote style and neither interpolates ("'a' . 'b'" ->
// "'ab'"). Merges across a newline are left alone.
type NoUselessConcatOperator struct {
	JuggleSimpleStrings bool
}

func (f NoUselessConcatOperator) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["juggle_simple_strings"].(bool); ok {
		f.JuggleSimpleStrings = v
	}
	return f
}

func (NoUselessConcatOperator) Name() string {
	return `PhpCsFixer\Fixer\Operator\NoUselessConcatOperatorFixer`
}

func (NoUselessConcatOperator) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NoUselessConcatOperatorFixer.php"
}

func (f NoUselessConcatOperator) Fix(s *tokens.Stream) bool {
	changed := false
	i := 0
	for i < s.Len() {
		t := s.At(i)
		if t.Kind != token.String {
			i++
			continue
		}
		q := stringQuote(t.Value)
		if q == 0 || (q == '"' && strings.Contains(t.Value, "$")) {
			i++
			continue
		}
		j := nextSignificantIndex(s, i)
		if j < 0 || s.At(j).Kind != token.Punct || s.At(j).Value != "." {
			i++
			continue
		}
		k := nextSignificantIndex(s, j)
		if k < 0 || s.At(k).Kind != token.String {
			i++
			continue
		}
		next := s.At(k).Value
		merged := t.Value[:len(t.Value)-1] + next[1:]
		if nq := stringQuote(next); nq != q || (q == '"' && strings.Contains(next, "$")) {
			simple, ok := noUselessConcatJuggle(f.JuggleSimpleStrings, t.Value, q, next, nq)
			if !ok {
				i++
				continue
			}
			merged = simple
		}
		if rangeHasNewline(s, i, k) {
			i++
			continue
		}
		s.SetValue(i, merged)
		for idx := k; idx > i; idx-- {
			s.RemoveAt(idx)
		}
		changed = true
		// stay at i so chained literals ("'a'.'b'.'c'") collapse fully
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/NoBinaryStringFixer.php
//
// NoBinaryString drops the binary-string prefix "b"/"B" (b"x" -> "x"). The
// lexer emits the prefix as a separate identifier directly before the string.
type NoBinaryString struct{}

func (NoBinaryString) Name() string {
	return `PhpCsFixer\Fixer\StringNotation\NoBinaryStringFixer`
}

func (NoBinaryString) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/NoBinaryStringFixer.php"
}

func (NoBinaryString) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Ident || (t.Value != "b" && t.Value != "B") {
			continue
		}
		if i+1 >= s.Len() || s.At(i+1).Kind != token.String {
			continue
		}
		if prev, ok := prevSignificant(s, i); ok {
			switch prev.Value {
			case "->", "?->", "::", `\`:
				continue
			}
		}
		s.RemoveAt(i)
		i--
		changed = true
	}
	return changed
}

// noUselessConcatJuggle merges a double-quoted string with a simple single-quoted
// one (either order) into a double-quoted string when juggle_simple_strings is on.
func noUselessConcatJuggle(enabled bool, first string, firstQuote byte, second string, secondQuote byte) (string, bool) {
	if !enabled || first == "" || second == "" {
		return "", false
	}
	var single string
	switch {
	case firstQuote == '"' && secondQuote == '\'' && !strings.Contains(first, "$"):
		single = second
	case firstQuote == '\'' && secondQuote == '"' && !strings.Contains(second, "$"):
		single = first
	default:
		return "", false
	}
	if strings.ContainsAny(single[1:len(single)-1], "$\"'\\") {
		return "", false
	}
	return `"` + first[1:len(first)-1] + second[1:len(second)-1] + `"`, true
}
